package gateway

import (
	"api/internal/handler"
	"api/internal/helper"
	"api/internal/namespace"
	rrmicro "api/internal/resolver/api"
	"api/stats"
	"fmt"
	"github.com/asim/go-micro/plugins/registry/consul/v3"
	"github.com/asim/go-micro/v3"
	ahandler "github.com/asim/go-micro/v3/api/handler"
	aapi "github.com/asim/go-micro/v3/api/handler/api"
	"github.com/asim/go-micro/v3/api/handler/event"
	ahttp "github.com/asim/go-micro/v3/api/handler/http"
	arpc "github.com/asim/go-micro/v3/api/handler/rpc"
	"github.com/asim/go-micro/v3/api/handler/web"
	"github.com/asim/go-micro/v3/api/resolver"
	"github.com/asim/go-micro/v3/api/resolver/grpc"
	"github.com/asim/go-micro/v3/api/resolver/host"
	"github.com/asim/go-micro/v3/api/resolver/path"
	"github.com/asim/go-micro/v3/api/router"
	"github.com/asim/go-micro/v3/api/server"
	"github.com/asim/go-micro/v3/api/server/acme"
	"github.com/asim/go-micro/v3/registry"

	//"github.com/asim/go-micro/v3/auth"
	//plugin "github.com/asim/go-micro/v3/plugins"
	regRouter "github.com/asim/go-micro/v3/api/router/registry"
	httpapi "github.com/asim/go-micro/v3/api/server/http"
	"github.com/asim/go-micro/v3/util/log"
	"github.com/gorilla/mux"
	"github.com/urfave/cli/v2"
	"net/http"
	"strings"
)

var (
	Name                  = "go.micro.api"
	Address               = ":8080"
	Handler               = "meta"
	Resolver              = "micro"
	RPCPath               = "/rpc"
	APIPath               = "/"
	ProxyPath             = "/{service:[a-zA-Z0-9]+}"
	Namespace             = "go.micro"
	Type                  = "api"
	HeaderPrefix          = "X-Micro-"
	EnableRPC             = false
	ACMEProvider          = "autocert"
	ACMEChallengeProvider = "cloudflare"
	ACMECA                = acme.LetsEncryptProductionCA
	RegistryAddress       = "127.0.0.1:8500"
)

func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	if len(ctx.String("server_name")) > 0 {
		Name = ctx.String("server_name")
	}
	if len(ctx.String("address")) > 0 {
		Address = ctx.String("address")
	}
	if len(ctx.String("handler")) > 0 {
		Handler = ctx.String("handler")
	}
	if len(ctx.String("resolver")) > 0 {
		Resolver = ctx.String("resolver")
	}
	if len(ctx.String("enable_rpc")) > 0 {
		EnableRPC = ctx.Bool("enable_rpc")
	}
	if len(ctx.String("acme_provider")) > 0 {
		ACMEProvider = ctx.String("acme_provider")
	}
	if len(ctx.String("type")) > 0 {
		Type = ctx.String("type")
	}
	if len(ctx.String("namespace")) > 0 {
		// remove the service type from the namespace to allow for
		// backwards compatability
		Namespace = strings.TrimSuffix(ctx.String("namespace"), "."+Type)
	}

	// apiNamespace has the format: "go.micro.api"
	//确定命名空间
	apiNamespace := Namespace + "." + Type

	// append name to opts
	//把名称加到 srv 选择项目中


	srvOpts = append(srvOpts, micro.Name(Name))


	if len(ctx.String("registry_address")) > 0 {
		RegistryAddress = ctx.String("registry_address")
	}

	var reOpt []registry.Option
	reOpt = append(reOpt, registry.Addrs(RegistryAddress))
	registry:=consul.NewRegistry(reOpt...)

	srvOpts = append(srvOpts, micro.Registry(registry))
	//初始化 registory

	// initialise service
	//初始化服务
	service := micro.NewService(srvOpts...)

	// Init plugins
	//初始化插件
	//for _, p := range Plugins() {
	//	p.Init(ctx)
	//}

	// Init API
	//初始化API
	var opts []server.Option

	//判断协议
	if ctx.Bool("enable_acme") {
		//hosts := helper.ACMEHosts(ctx)
		//opts = append(opts, server.EnableACME(true))
		//opts = append(opts, server.ACMEHosts(hosts...))
		//switch ACMEProvider {
		//case "autocert":
		//	opts = append(opts, server.ACMEProvider(autocert.NewProvider()))
		//case "certmagic":
		//	if ACMEChallengeProvider != "cloudflare" {
		//		log.Fatal("The only implemented DNS challenge provider is cloudflare")
		//	}
		//
		//	apiToken := os.Getenv("CF_API_TOKEN")
		//	if len(apiToken) == 0 {
		//		log.Fatal("env variables CF_API_TOKEN and CF_ACCOUNT_ID must be set")
		//	}
		//
		//	storage := certmagic.NewStorage(
		//		memory.NewSync(),
		//		service.Options().Store,
		//	)
		//
		//	config := cloudflare.NewDefaultConfig()
		//	config.AuthToken = apiToken
		//	config.ZoneToken = apiToken
		//	challengeProvider, err := cloudflare.NewDNSProviderConfig(config)
		//	if err != nil {
		//		log.Fatal(err.Error())
		//	}
		//
		//	opts = append(opts,
		//		server.ACMEProvider(
		//			certmagic.NewProvider(
		//				acme.AcceptToS(true),
		//				acme.CA(ACMECA),
		//				acme.Cache(storage),
		//				acme.ChallengeProvider(challengeProvider),
		//				acme.OnDemand(false),
		//			),
		//		),
		//	)
		//default:
		//	log.Fatalf("%s is not a valid ACME provider\n", ACMEProvider)
		//}
	} else if ctx.Bool("enable_tls") {
		config, err := helper.TLSConfig(ctx)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		opts = append(opts, server.EnableTLS(true))
		opts = append(opts, server.TLSConfig(config))
	}

	if ctx.Bool("enable_cors") {
		opts = append(opts, server.EnableCORS(true))
	}

	// create the router
	//创建路由
	var h http.Handler
	r := mux.NewRouter()
	h = r

	//提供监控接口
	if ctx.Bool("enable_stats") {
		st := stats.New()
		r.HandleFunc("/stats", st.StatsHandler)
		h = st.ServeHTTP(r)
		st.Start()
		defer st.Stop()
	}

	// return version and list of services
	//返回版本和服务
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			return
		}
		response := fmt.Sprintf(`{"version": "%s"}`, ctx.App.Version)
		w.Write([]byte(response))
	})

	// strip favicon.ico
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})

	// register rpc handler
	//注册 rpc
	if EnableRPC {
		log.Infof("Registering RPC Handler at %s", RPCPath)
		r.HandleFunc(RPCPath, handler.RPC)
	}

	// create the namespace resolver
	nsResolver := namespace.NewResolver(Type, Namespace)

	// resolver options
	ropts := []resolver.Option{
		resolver.WithNamespace(nsResolver.ResolveWithType),
		resolver.WithHandler(Handler),
	}

	// default resolver

	rr := rrmicro.NewResolver(ropts...)

	switch Resolver {
	case "host":
		rr = host.NewResolver(ropts...)
	case "path":
		rr = path.NewResolver(ropts...)
	case "grpc":
		rr = grpc.NewResolver(ropts...)
	}

	switch Handler {
	case "rpc":
		log.Infof("Registering API RPC Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(arpc.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		rp := arpc.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(rp)
	case "api":
		log.Infof("Registering API Request Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(aapi.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		ap := aapi.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(ap)
	case "event":
		log.Infof("Registering API Event Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(event.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		ev := event.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(ev)
	case "http", "proxy":
		log.Infof("Registering API HTTP Handler at %s", ProxyPath)
		rt := regRouter.NewRouter(
			router.WithHandler(ahttp.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		ht := ahttp.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(ProxyPath).Handler(ht)
	case "web":
		log.Infof("Registering API Web Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(web.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		w := web.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(w)
	default:
		log.Infof("Registering API Default Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		r.PathPrefix(APIPath).Handler(handler.Meta(service, rt, nsResolver.ResolveWithType))
	}

	// reverse wrap handler
	//plugins := append(Plugins(), plugin.Plugins()...)
	//for i := len(plugins); i > 0; i-- {
	//	h = plugins[i-1].Handler()(h)
	//}

	// create the auth wrapper and the server
	//authWrapper := auth.Wrapper(rr, nsResolver)

	api := httpapi.NewServer(Address)

	api.Init(opts...)
	api.Handle("/", h)

	// Start API
	if err := api.Start(); err != nil {
		log.Fatal(err)
	}

	// Run server
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}

	// Stop API
	if err := api.Stop(); err != nil {
		log.Fatal(err)
	}
}

func Commands(options ...micro.Option) []*cli.Command {
	command := &cli.Command{
		Name:  "api",
		Usage: "Run the api gateway",
		Action: func(ctx *cli.Context) error {
			Run(ctx, options...)
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Set the api address e.g 0.0.0.0:8080",
				EnvVars: []string{"MICRO_API_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "handler",
				Usage:   "Specify the request handler to be used for mapping HTTP requests to services; {api, event, http, rpc}",
				EnvVars: []string{"MICRO_API_HANDLER"},
			},
			&cli.StringFlag{
				Name:    "namespace",
				Usage:   "Set the namespace used by the API e.g. com.example",
				EnvVars: []string{"MICRO_API_NAMESPACE"},
			},
			&cli.StringFlag{
				Name:    "type",
				Usage:   "Set the service type used by the API e.g. api",
				EnvVars: []string{"MICRO_API_TYPE"},
			},
			&cli.StringFlag{
				Name:    "resolver",
				Usage:   "Set the hostname resolver used by the API {host, path, grpc}",
				EnvVars: []string{"MICRO_API_RESOLVER"},
			},
			&cli.BoolFlag{
				Name:    "enable_rpc",
				Usage:   "Enable call the backend directly via /rpc",
				EnvVars: []string{"MICRO_API_ENABLE_RPC"},
			},
			&cli.BoolFlag{
				Name:    "enable_cors",
				Usage:   "Enable CORS, allowing the API to be called by frontend applications",
				EnvVars: []string{"MICRO_API_ENABLE_CORS"},
				Value:   true,
			},
		},
	}

	//for _, p := range plugin.Plugin{}() {
	//	if cmds := p.Commands(); len(cmds) > 0 {
	//		command.Subcommands = append(command.Subcommands, cmds...)
	//	}
	//
	//	if flags := p.Flags(); len(flags) > 0 {
	//		command.Flags = append(command.Flags, flags...)
	//	}
	//}

	return []*cli.Command{command}
}

