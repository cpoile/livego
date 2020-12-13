module github.com/cpoile/livego

go 1.15

require (
	github.com/auth0/go-jwt-middleware v0.0.0-20190805220309-36081240882b
	github.com/cpoile/aws-test v0.1.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.4.7
	github.com/giorgisio/goav v0.1.0
	github.com/go-redis/redis/v7 v7.2.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/kr/pretty v0.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.6.3
	github.com/stretchr/testify v1.6.1
	github.com/urfave/negroni v1.0.0 // indirect
)

replace github.com/giorgisio/goav => /Users/chris/go/src/github.com/goav

replace github.com/cpoile/aws-test => /Users/chris/go/src/github.com/cpoile/aws-test
