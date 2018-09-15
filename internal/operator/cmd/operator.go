package cmd

import (
	"log"
	"time"

	"github.com/jelmersnoeck/grafana-operator/internal/kit/signals"
	"github.com/jelmersnoeck/grafana-operator/internal/operator"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var operatorFlags struct {
	MasterURL    string
	KubeConfig   string
	ResyncPeriod string

	MetricsAddr string
	MetricsPort int
}

// operatorCmd represents the operator command
var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Run the Grafana Operator",
	Run:   runOperator,
}

func runOperator(cmd *cobra.Command, args []string) {
	stopCh := signals.SetupSignalHandler()

	resync, err := time.ParseDuration(operatorFlags.ResyncPeriod)
	if err != nil {
		log.Fatalf("Error parsing ResyncPeriod: %s", err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags(operatorFlags.MasterURL, operatorFlags.KubeConfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building Kubernetes clientset: %s", err)
	}

	gClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building Grafana clientset: %s", err)
	}

	op, err := operator.NewOperator(kubeClient, gClient, resync)
	if err != nil {
		log.Fatalf("Error building Grafana Operator: %s", err)
	}

	if err := op.Run(stopCh); err != nil {
		log.Fatalf("Error running the operator: %s", err)
	}
}

func init() {
	rootCmd.AddCommand(operatorCmd)

	operatorCmd.PersistentFlags().StringVar(&operatorFlags.MasterURL, "master-url", "", "The URL of the master API.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.KubeConfig, "kubeconfig", "", "Kubeconfig which should be used to talk to the API.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.ResyncPeriod, "resync-period", "30s", "Resyncing period to ensure all monitors are up to date.")

	operatorCmd.PersistentFlags().StringVar(&operatorFlags.MetricsAddr, "metrics-addr", "0.0.0.0", "address the metrics server will bind to")
	operatorCmd.PersistentFlags().IntVar(&operatorFlags.MetricsPort, "metrics-port", 9090, "port on which the metrics server is available")
}
