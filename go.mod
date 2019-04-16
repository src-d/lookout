module github.com/src-d/lookout

go 1.12

require (
	github.com/BurntSushi/toml v0.3.0
	github.com/Masterminds/squirrel v0.0.0-20170825200431-a6b93000bd21
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7
	github.com/bradleyfalzon/ghinstallation v0.1.2
	github.com/davecgh/go-spew v1.1.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emirpasic/gods v1.9.0
	github.com/go-chi/chi v3.3.3+incompatible
	github.com/gogo/protobuf v1.2.1
	github.com/golang-migrate/migrate v3.2.0+incompatible
	github.com/golang/protobuf v0.0.0-20180724203048-93b26e6a70e3
	github.com/google/btree v0.0.0-20180124185431-e89373fe6b4a
	github.com/google/go-github/v24 v24.0.1
	github.com/google/go-querystring v0.0.0-20170111101155-53e6ce116135
	github.com/gorilla/context v1.1.1
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.1.3
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/hashicorp/golang-lru v0.5.1
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99
	github.com/jessevdk/go-flags v1.4.0
	github.com/jinzhu/copier v0.0.0-20180308034124-7e38e58719c3
	github.com/jpillora/backoff v0.0.0-20170918002102-8eab2debe79d
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/kevinburke/ssh_config v0.0.0-20180317175531-9fc7bb800b55
	github.com/konsorten/go-windows-terminal-sequences v1.0.1
	github.com/lann/builder v0.0.0-20180216234317-1b87b36280d0
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0
	github.com/lib/pq v0.0.0-20180523175426-90697d60dd84
	github.com/mattn/go-colorable v0.0.9
	github.com/mattn/go-isatty v0.0.3
	github.com/mcuadros/go-lookup v0.0.0-20171110082742-5650f26be767
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/mitchellh/go-homedir v0.0.0-20180523094522-3864e76763d9
	github.com/mwitkow/grpc-proxy v0.0.0-20181017164139-0f1106ef9c76
	github.com/oklog/ulid v1.0.0
	github.com/pelletier/go-buffruneio v0.2.0
	github.com/petar/GoLLRB v0.0.0-20130427215148-53be0d36a84c
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.8.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/rs/cors v1.6.0
	github.com/sanity-io/litter v0.0.0-20180714121731-09e3a73d5b65
	github.com/satori/go.uuid v0.0.0-20180103174451-36e9d2ebbde5
	github.com/sergi/go-diff v1.0.0
	github.com/sirupsen/logrus v1.1.1
	github.com/src-d/envconfig v1.0.0
	github.com/src-d/gcfg v1.3.0
	github.com/src-d/go-oniguruma v1.0.0
	github.com/src-d/lookout-test-fixtures v0.0.0-20190402142344-11bd37726868
	github.com/streadway/amqp v0.0.0-20180806233856-70e15c650864
	github.com/stretchr/objx v0.1.1
	github.com/stretchr/testify v1.3.0
	github.com/toqueteos/trie v0.0.0-20150530104557-56fed4a05683
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/xanzy/ssh-agent v0.1.0
	golang.org/x/crypto v0.0.0-20180621125126-a49355c7e3f8
	golang.org/x/net v0.0.0-20180621144259-afe8f62b1d6b
	golang.org/x/oauth2 v0.0.0-20181203162652-d668ce993890
	golang.org/x/sys v0.0.0-20180627142611-7138fd3d9dc8
	golang.org/x/text v0.3.0
	google.golang.org/appengine v1.2.0
	google.golang.org/genproto v0.0.0-20180621235812-80063a038e33
	google.golang.org/grpc v1.19.1
	gopkg.in/bblfsh/sdk.v1 v1.17.0
	gopkg.in/check.v1 v1.0.0-20161208181325-20d25e280405
	gopkg.in/src-d/enry.v1 v1.7.2
	gopkg.in/src-d/go-billy.v4 v4.2.0
	gopkg.in/src-d/go-cli.v0 v0.0.0-20181105080154-d492247bbc0d
	gopkg.in/src-d/go-errors.v0 v0.1.0
	gopkg.in/src-d/go-errors.v1 v1.0.0
	gopkg.in/src-d/go-git-fixtures.v3 v3.3.0
	gopkg.in/src-d/go-git.v4 v4.10.0
	gopkg.in/src-d/go-kallax.v1 v1.3.5
	gopkg.in/src-d/go-log.v1 v1.0.1
	gopkg.in/src-d/go-queue.v1 v1.0.6
	gopkg.in/src-d/lookout-sdk.v0 v0.6.2
	gopkg.in/toqueteos/substring.v1 v1.0.2
	gopkg.in/vmihailenco/msgpack.v2 v2.9.1
	gopkg.in/warnings.v0 v0.1.2
	gopkg.in/yaml.v2 v2.2.2
)
