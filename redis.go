package main

// db 0: caching of upstream requests
// {
//     "id": "...",
//     "name": "test",
//     "region": "US",
//     "age": 100
// }

// db 1: configuration for routes/upstreams and auth methods
// {
//      Path:      "/api/example",
// 		TargetURL: "http://localhost:8081/hello",
// 		Middleware: []Middleware{
// 		    logConfig.LogHandler,
// 			CORS,
// 			MethodMiddleware([]string{"GET", "POST"}),
// 		},
// 	},
