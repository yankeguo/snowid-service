package ufx

import "go.uber.org/fx"

var Module = fx.Module(
	"ufx",
	fx.Provide(
		ProberParamsFromConf,
		RouterParamsFromConf,
		ServerParamsFromConf,
		NewProber,
		NewRouter,
		NewServer,
	),
	fx.Invoke(SetupOTEL),
	fx.Invoke(func(Server) {}),
)
