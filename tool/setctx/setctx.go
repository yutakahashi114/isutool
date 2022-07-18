// 適用したいファイルがあるディレクトリから実行する
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	l := &loader.Config{ParserMode: parser.ParseComments}
	f, err := os.Open("go.mod")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	pkgName := strings.TrimPrefix(scanner.Text(), "module ")

	dirPkgs := [][2]string{
		{".", pkgName},
	}
	// パッケージ毎にgoファイルを読み込む
	for _, dirPkg := range dirPkgs {
		files, err := os.ReadDir(dirPkg[0])
		if err != nil {
			return err
		}
		astfs := make([]*ast.File, 0)
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if !strings.HasSuffix(f.Name(), ".go") {
				continue
			}
			astf, err := l.ParseFile(dirPkg[0]+"/"+f.Name(), nil)
			if err != nil {
				return err
			}
			astfs = append(astfs, astf)
		}
		l.CreateFromFiles(dirPkg[1], astfs...)
	}

	prog, err := l.Load()
	if err != nil {
		return err
	}

	// 変換する関数を抽出する
	// 以下は機械的に変換できないで最初からセットしておく
	var ctxFuncMap = map[string]string{
		"Begin":     "BeginTx",
		"Beginx":    "BeginTxx",
		"MustBegin": "MustBeginTx",
	}
	for sqlPkgName := range dbPkgNameMap {
		pkg := prog.Package(sqlPkgName)
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if !fd.Name.IsExported() {
					continue
				}
				// 引数の0番目に"context.Context"を持つか確認
				typ := fd.Type
				if len(typ.Params.List) < 1 {
					continue
				}
				arg0 := typ.Params.List[0]
				se, ok := arg0.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				if se.Sel.Name != "Context" {
					continue
				}
				ident, ok := se.X.(*ast.Ident)
				if !ok {
					continue
				}
				if ident.Name != "context" {
					continue
				}
				// 関数名のSuffixの"Context"を削除する
				if strings.HasSuffix(fd.Name.Name, "Context") {
					ctxFuncMap[strings.TrimSuffix(fd.Name.Name, "Context")] = fd.Name.Name
				}
			}
		}
	}

	dbPkgMap := make(map[*types.Package]struct{})
	for pkgName, m := range dbPkgNameMap {
		pkg := prog.Package(pkgName)
		if pkg == nil {
			continue
		}
		dbPkgMap[pkg.Pkg] = m
	}

	// DBのメソッドの変換が必要な関数を抽出する
	dbCallFuncMap := map[*ast.FuncDecl]struct{}{}
	targetPkgs := make([]*loader.PackageInfo, 0, len(dirPkgs))
	for _, dirPkg := range dirPkgs {
		targetPkgs = append(targetPkgs, prog.Package(dirPkg[1]))
	}
	for _, dirPkg := range dirPkgs {
		pkg := prog.Package(dirPkg[1])
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok {
					continue
				}
				found := false
				visited := map[ast.Node]struct{}{}
				var findSQLCall func(n ast.Node) bool
				findSQLCall = func(n ast.Node) bool {
					if _, ok := visited[n]; ok {
						return false
					}
					visited[n] = struct{}{}
					if ce, ok := n.(*ast.CallExpr); ok {
						var callObj *ast.Object
						var callIdent *ast.Ident
						switch expr := ce.Fun.(type) {
						case *ast.SelectorExpr:
							// (メソッド or 外部パッケージの関数) の実行
							if isSQLCall(targetPkgs, dbPkgMap, ctxFuncMap, expr) {
								found = true
								return false
							}
							callObj = expr.Sel.Obj
							callIdent = expr.Sel
						case *ast.Ident:
							// 関数の実行
							callObj = expr.Obj
							callIdent = expr
						}
						if callObj == nil {
							// 別パッケージで定義した関数を使用している場合
							fd := getFuncDeclByIdent(callIdent, targetPkgs)
							if fd != nil {
								ast.Inspect(fd.Body, findSQLCall)
							}
						} else {
							if fd, ok := callObj.Decl.(*ast.FuncDecl); ok {
								ast.Inspect(fd, findSQLCall)
							}
						}
					}
					return true
				}
				ast.Inspect(fd.Body, findSQLCall)
				if found {
					dbCallFuncMap[fd] = struct{}{}
				}
			}
		}
	}

	// 変換する
	for _, dirPkg := range dirPkgs {
		pkg := prog.Package(dirPkg[1])
		for _, f := range pkg.Files {
			shouldAddCtxToPkg := false
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok {
					continue
				}
				typ := fd.Type
				// 関数の引数からcontext.Contextオブジェクトの名前を特定する
				ctxValueName := getCtxArgName(typ.Params.List)
				hasNotCtxArg := false
				if ctxValueName == "" {
					hasNotCtxArg = true
					if fd.Name.Name == "main" {
						// main関数はcontext.Background()を使う
						ctxValueName = "context.Background()"
					} else {
						ctxValueName = "ctx"
					}
				}
				shouldAddCtxToArg := false

				// ast.Inspectの引数に使う
				convert := func(ctxValueName string) func(ast.Node) bool {
					return func(node ast.Node) bool {
						if ce, ok := node.(*ast.CallExpr); ok {
							var callObj *ast.Object
							var callIdent *ast.Ident
							switch expr := ce.Fun.(type) {
							case *ast.SelectorExpr:
								// (メソッド or 外部パッケージの関数) の実行
								if _, ok := dbPkgMap[pkg.Info.ObjectOf(expr.Sel).Pkg()]; ok {
									if ctxFuncName, ok := ctxFuncMap[expr.Sel.Name]; ok {
										shouldAddCtxToArg = true
										// SQL実行関数をctx対応に変換する
										fmt.Printf("%d %s -> %s\n", prog.Fset.Position(node.Pos()).Line, expr.Sel.Name, ctxFuncName)
										expr.Sel.Name = ctxFuncName
										// 引数にctxを追加する
										ce.Args = append([]ast.Expr{&ast.Ident{Name: ctxValueName}}, ce.Args...)
										// 特定の関数の場合、引数にnilを追加する
										if ctxFuncName == "BeginTxx" || ctxFuncName == "BeginTx" || ctxFuncName == "MustBeginTx" {
											ce.Args = append(ce.Args, &ast.Ident{Name: "nil"})
										}
									} else {
										// sqlパッケージの関数で変換しなかったものを出力しておく
										fmt.Printf("%d %s\n", prog.Fset.Position(node.Pos()).Line, expr.Sel.Name)
									}
									return true
								}
								callObj = expr.Sel.Obj
								callIdent = expr.Sel
							case *ast.Ident:
								// 関数の実行
								callObj = expr.Obj
								callIdent = expr
							}
							if callObj == nil {
								fd := getFuncDeclByIdent(callIdent, targetPkgs)
								if fd != nil {
									// SQL実行関数を内部で呼んでいる関数の引数にctxを追加する
									if _, ok := dbCallFuncMap[fd]; ok {
										if !hasCtxArg(pkg.Info, ce.Args) {
											shouldAddCtxToArg = true
											ce.Args = append([]ast.Expr{&ast.Ident{Name: ctxValueName}}, ce.Args...)
										}
									}
								}
							} else {
								if fd, ok := callObj.Decl.(*ast.FuncDecl); ok {
									// SQL実行関数を内部で呼んでいる関数の引数にctxを追加する
									if _, ok := dbCallFuncMap[fd]; ok {
										if !hasCtxArg(pkg.Info, ce.Args) {
											shouldAddCtxToArg = true
											ce.Args = append([]ast.Expr{&ast.Ident{Name: ctxValueName}}, ce.Args...)
										}
									}
								}
							}
						}
						return true
					}
				}

				ast.Inspect(fd, func(node ast.Node) bool {
					if fl, ok := node.(*ast.FuncLit); ok {
						// 無名関数の引数にctxがあれば、そちらを優先する
						ctxValueNameInFuncLit := getCtxArgName(fl.Type.Params.List)
						if ctxValueNameInFuncLit == "" {
							ctxValueNameInFuncLit = ctxValueName
						}
						ast.Inspect(fl, convert(ctxValueNameInFuncLit))
						return false
					}
					return convert(ctxValueName)(node)
				})

				if shouldAddCtxToArg && hasNotCtxArg {
					// 引数にcontext.Contextオブジェクトがない場合は追加する
					shouldAddCtxToPkg = true
					if fd.Name.Name != "main" {
						typ.Params.List = append([]*ast.Field{{
							Names: []*ast.Ident{{Name: "ctx"}},
							Type: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "context"},
								Sel: &ast.Ident{Name: "Context"},
							},
						}}, typ.Params.List...)
					}
				}
			}
			// contextパッケージを追加する
			if shouldAddCtxToPkg {
				containCtxPkg := false
				for _, importPkg := range f.Imports {
					if importPkg.Path.Value == `"context"` {
						containCtxPkg = true
					}
				}
				if !containCtxPkg {
					astutil.AddImport(prog.Fset, f, "context")
				}
			}
			// continue
			// ファイルを更新する
			file, err := os.Create(prog.Fset.File(f.Pos()).Name())
			if err != nil {
				return err
			}

			pp := &printer.Config{Tabwidth: 8, Mode: printer.UseSpaces | printer.TabIndent}
			pp.Fprint(file, prog.Fset, f)
		}
	}
	return nil
}

var dbPkgNameMap = map[string]struct{}{
	"database/sql":            {},
	"github.com/jmoiron/sqlx": {},
}

func isSQLCall(targetPkgs []*loader.PackageInfo, pkgMap map[*types.Package]struct{}, ctxFuncMap map[string]string, se *ast.SelectorExpr) bool {
	for _, targetPkg := range targetPkgs {
		obj := targetPkg.Info.ObjectOf(se.Sel)
		if obj == nil {
			continue
		}
		if _, ok := pkgMap[obj.Pkg()]; ok {
			if _, ok := ctxFuncMap[se.Sel.Name]; ok {
				return true
			}
		}
	}
	return false
}

var ctxArgMap = map[string]string{
	"context.Context": "",
	"echo.Context":    ".Request().Context()",
	"http.Request":    ".Context()",
}

func getCtxArgName(fields []*ast.Field) string {
	for _, arg := range fields {
		typ := getFieldTypeString(arg.Type)
		ctxValue, ok := ctxArgMap[typ]
		if !ok {
			continue
		}
		return arg.Names[0].Name + ctxValue
	}
	return ""
}

func getFieldTypeString(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.SelectorExpr:
		ident, ok := expr.X.(*ast.Ident)
		if !ok {
			return ""
		}
		return ident.Name + "." + expr.Sel.Name
	case *ast.StarExpr:
		return getFieldTypeString(expr.X)
	}
	return ""
}

func hasCtxArg(info types.Info, args []ast.Expr) bool {
	for _, arg := range args {
		typ := getArgTypeString(info, arg)
		if typ == "" {
			continue
		}
		_, ok := ctxArgMap[typ]
		if !ok {
			continue
		}
		return true
	}
	return false
}

func getArgTypeString(info types.Info, expr ast.Expr) string {
	switch arg := expr.(type) {
	case *ast.Ident:
		typ := info.TypeOf(arg)
		if typ == nil {
			return ""
		}
		n := getTypeString(typ)
		if n != "" {
			return n
		}
	case *ast.CallExpr:
		switch expr := arg.Fun.(type) {
		case *ast.SelectorExpr:
			return getArgTypeString(info, expr.X)
		case *ast.Ident:
			if fd, ok := expr.Obj.Decl.(*ast.FuncDecl); ok {
				if len(fd.Type.Results.List) == 1 {
					return getArgTypeString(info, fd.Type.Results.List[0].Type)
				}
			}
		}
	}
	return ""
}

func getTypeString(typ types.Type) string {
	switch typ := typ.(type) {
	case *types.Named:
		return typ.Obj().Pkg().Name() + "." + typ.Obj().Name()
	case *types.Pointer:
		return getTypeString(typ.Elem())
	}
	return ""
}

func getFuncDeclByIdent(ident *ast.Ident, targetPkgs []*loader.PackageInfo) *ast.FuncDecl {
	var obj types.Object
	for _, targetPkg := range targetPkgs {
		obj = targetPkg.Info.ObjectOf(ident)
		if obj != nil {
			break
		}
	}
	if obj == nil {
		return nil
	}
	for _, targetPkg := range targetPkgs {
		// 定義されたパッケージ内で一致する関数を探す
		if targetPkg.Pkg == obj.Pkg() {
			for _, f := range targetPkg.Files {
				for _, d := range f.Decls {
					fd, ok := d.(*ast.FuncDecl)
					if !ok {
						continue
					}
					if obj == targetPkg.Info.ObjectOf(fd.Name) {
						return fd
					}
				}
			}
			break
		}
	}
	return nil
}
