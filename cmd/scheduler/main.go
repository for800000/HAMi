/*
 * Copyright © 2021 peizhaoyou <peizhaoyou@4paradigm.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
    "4pd.io/k8s-vgpu/pkg/util"
    "net"
    "net/http"

    pb "4pd.io/k8s-vgpu/pkg/api"
    "4pd.io/k8s-vgpu/pkg/scheduler/config"
    "4pd.io/k8s-vgpu/pkg/scheduler/routes"
    "4pd.io/k8s-vgpu/pkg/scheduler/service"
    "github.com/julienschmidt/httprouter"
    "github.com/spf13/cobra"
    "google.golang.org/grpc"
    "k8s.io/klog/v2"
)

//var version string

var (
    tlsKeyFile string
    tlsCertFile string
    rootCmd = &cobra.Command{
        Use:   "scheduler",
        Short: "kubernetes vgpu scheduler",
        Run: func(cmd *cobra.Command, args []string) {
            start()
        },
    }
)
func init() {
    rootCmd.Flags().SortFlags = false
    rootCmd.PersistentFlags().SortFlags = false

    rootCmd.Flags().StringVar(&config.GrpcBind, "grpc_bind", "127.0.0.1:9090", "grpc server bind address")
    rootCmd.Flags().StringVar(&config.HttpBind, "http_bind", "127.0.0.1:8080", "http server bind address")
    rootCmd.Flags().StringVar(&tlsCertFile, "cert_file", "", "tls cert file")
    rootCmd.Flags().StringVar(&tlsKeyFile, "key_file", "", "tls key file")

    rootCmd.PersistentFlags().AddGoFlagSet(util.GlobalFlagSet())
}

func start()  {
    deviceService := service.NewDeviceService()
    scheduler := service.NewScheduler(deviceService)
    scheduler.Start()
    defer scheduler.Stop()

    // start grpc server
    lisGrpc, _ := net.Listen("tcp", config.GrpcBind)
    defer lisGrpc.Close()
    s := grpc.NewServer()
    pb.RegisterDeviceServiceServer(s, deviceService)
    go func() {
        err := s.Serve(lisGrpc)
        if err != nil {
            klog.Fatal(err)
        }
    }()

    // start http server
    router := httprouter.New()
    router.POST("/filter", routes.PredicateRoute(scheduler))
    router.POST("/webhook", routes.WebHookRoute())
    klog.Info("listen on ", config.HttpBind)
    if len(tlsCertFile) == 0 || len(tlsKeyFile) == 0 {
        if err := http.ListenAndServe(config.HttpBind, router); err != nil {
            klog.Fatal("Listen and Serve error, ", err)
        }
    } else {
        if err := http.ListenAndServeTLS(config.HttpBind, tlsCertFile, tlsKeyFile, router); err != nil {
            klog.Fatal("Listen and Serve error, ", err)
        }
    }
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        klog.Fatal(err)
    }
}
