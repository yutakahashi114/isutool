* library
    * async
        * 引数の関数を非同期で実行し、後で結果とエラーを取得する
    * asyncexecute
        * キューに詰めて非同期の無限ループで実行する
    * btreemutex
        * [`github.com/google/btree`](https://github.com/google/btree) をmutexで囲ったもの
    * bulkexecute
        * asyncexecuteの同期版
    * cache
        * mapをmutexで囲ったもの
    * dataloader
        * リクエストを跨いで処理を一本化する
    * index
        * [`github.com/google/btree`](https://github.com/google/btree) を使い、複数の条件でソートする
    * mmap
        * readしかできないがメモリを消費しないmapを作る
    * mutexmap
        * キーごとにmutexでロックする
    * trace
        * エンドポイントごとのSQL実行状況を計測したり、pprofをとったり、explainをとったり
* tool
    * encodegen
        * 構造体から `[]byte` のエンコード、デコードするメソッドを生やす
        * jsonやgobより数倍速い
    * setctx
        * `database/sql` や [`github.com/jmoiron/sqlx`](https://github.com/jmoiron/sqlx) のメソッドをctx対応のものに差し替える
        * たまに失敗してコンパイルエラーになるので適宜修正する

