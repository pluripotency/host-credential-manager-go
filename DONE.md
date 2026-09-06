## やりたいこと
- ./hcm-client/README.mdを作成し、プログラムの役割、ビルド方法、詳細な使用方法を書いてください。
- ./hcm-client/built/README.mdを作成し、builtされたワンバイナリを使用する基本的使用方法を書いてください。
- ./hcm-client/built/run.shを作成し、SERVER_IP=127.0.0.1としたbashによる起動スクリプトを用意してください。
- ./hcm-client/build.shは./hcm-client/built/にcert、hcm-clientワンバイナリを出力するようにし、この２つは.gitignoreに追加してください。
- hcm-clientはhost-credential-manager-goのログイン後のページトップでビルド済み./hcm-client/builtをhcm-client.tgzとしてダウンロードできるようにして。ビルド済みでない場合はビルドをするようにして。

## やりたいこと
- host-credential-manager-goはssh,telnet等のcliで接続するノードの管理という点で、hcm-clientを使用してssh, telnet接続できるようにしたいのですが、現状SSHPASSやfzfやpythonの制限等があって扱いにくいです。goplur/example/embedを使用すればSSHPASS/fzfへの依存なく、ワンバイナリとして配布できるためこれを使用した方法に変えたいです。
- ./hcm-clientディレクトリを作ってgoplur/example/embedをベースとしたクライアントを作成して
- go版hcm-clientはcert/cacert.pemがある場合はこれを含んでワンバイナリにしたいです。
- host-credential-manager-go側では、telnetを使用するnetwork機器、sshを使用するAlmaLinux9をOSとしたLinux等のデフォルト設定を作成して、ssh/telnetのクライアントをhcm-clientで取得できるようにしてほしいです。

## DONE
- [x] 上のやりたいことのgo版hcm_clientを実装して実行可能にして
