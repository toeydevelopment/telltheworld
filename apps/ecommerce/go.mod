module github.com/toeydevelopment/telltheworld/apps/ecommerce

replace github.com/toeydevelopment/telltheworld/apis/teller => ../../apis/teller

go 1.25

require (
	github.com/go-kratos/kratos/v2 v2.9.1
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.7.0
	github.com/samber/lo v1.52.0
	github.com/toeydevelopment/telltheworld/apis/ecommerce v0.0.0
	go.uber.org/automaxprocs v1.6.0
	google.golang.org/protobuf v1.36.10
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-kratos/aegis v0.2.0 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/grpc v1.76.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace telltheworld/apis/teller => ../../apis/teller

replace github.com/toeydevelopment/telltheworld/apis/ecommerce => ../../apis/ecommerce

replace telltheworld/internal => ../../internal

replace telltheworld/pkg => ../../pkg
