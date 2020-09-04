package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/jacobweinstock/ovaimporter/pkg/vsphere"

	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	buildTime                     string
	gitCommit                     string
	cfgFile                       string
	url                           string
	user                          string
	password                      string
	datacenter                    string
	ova                           string
	folder                        string
	network                       string
	datastore                     string
	timeout                       int
	responseFileDirectory         string
	responseFileName              = "response.json"
	responseFileDirectoryFallback = "./"

	rootCmd = &cobra.Command{
		Use:     appName,
		Short:   "import an ova into a vcenter",
		Long:    fmt.Sprintf("%v is a CLI library that imports a remote ova into a vcenter.", appName),
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			var importOva importerResponse
			err := importOva.run()
			importOva.response(err)
		},
	}
)

var (
	appInfo = versionResponse{
		Version:   version,
		Name:      appName,
		GitCommit: gitCommit,
		Built:     buildTime,
	}
)

type versionResponse struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	GitCommit string `json:"gitCommit"`
	Built     string `json:"built"`
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initLogging)
	initLogging()
	initConfig()
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/.%v.yaml)", appName))
	rootCmd.PersistentFlags().StringVar(&responseFileDirectory, "dir", "./", "directory to write response file")
	rootCmd.PersistentFlags().StringVar(&url, "url", "", "vCenter url")
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "vCenter username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "vCenter password")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 5, "timeout")
	rootCmd.PersistentFlags().StringVar(&ova, "ova", "", "local file or remote URL of an OVA to import")
	rootCmd.PersistentFlags().StringVar(&folder, "folder", "", "folder into which to upload the OVA (example vm/my/folder)")
	rootCmd.PersistentFlags().StringVar(&network, "network", "", "network to attach to the template")
	rootCmd.PersistentFlags().StringVar(&datastore, "datastore", "", "vCenter datastore to which to upload the OVA")
	rootCmd.PersistentFlags().StringVar(&datacenter, "datacenter", "", "vCenter datacenter name")
	_ = rootCmd.MarkPersistentFlagRequired("ova")
	_ = rootCmd.MarkPersistentFlagRequired("url")
	_ = rootCmd.MarkPersistentFlagRequired("user")
	_ = rootCmd.MarkPersistentFlagRequired("password")
	info, _ := json.Marshal(appInfo)
	rootCmd.SetVersionTemplate(string(info))
}

func (i *importerResponse) run() error {
	var err error
	tout := time.Duration(timeout) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), tout)
	defer cancel()

	client, err := vsphere.NewClient(ctx, url, user, password)
	if err != nil {
		return err
	}
	client.Datacenter, err = client.GetDatacenterOrDefault(datacenter)
	if err != nil {
		return err
	}
	client.Network, err = client.GetNetworkOrDefault(network)
	if err != nil {
		return err
	}
	client.Datastore, err = client.GetDatastoreOrDefault(datastore)
	if err != nil {
		return err
	}
	// resource pool is need for the upload but doesnt really matter so we use the default
	client.ResourcePool, err = client.GetResourcePoolOrDefault("")
	if err != nil {
		return err
	}
	client.Folder, err = client.GetFolderOrDefault(folder)
	if err != nil {
		return err
	}

	info, err := client.DeployOVATemplate(ova)
	if err != nil {
		return err
	}
	i.AlreadyExists = info.AlreadyExists
	i.Name = info.TemplateName
	i.Success = true
	return err
}

func (i *importerResponse) response(err error) {
	r := i.ToLogrusFields()
	r["responseFile"] = path.Join(responseFileDirectory, responseFileName)
	if err != nil {
		r["errorMsg"] = err.Error()
		log.WithFields(r).Fatal()
	}
	log.WithFields(r).Info()
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			er(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName("." + appName)
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix(appName)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	postInitCommands([]*cobra.Command{rootCmd})
}

func initLogging() {
	log.SetFormatter(&log.JSONFormatter{})

	if responseFileDirectory == "./" {
		curDir, err := os.Getwd()
		if err == nil {
			responseFileDirectory = path.Join(curDir, responseFileDirectory)
		}
	}

	if _, err := os.Stat(responseFileDirectory); os.IsNotExist(err) {
		err := os.Mkdir(responseFileDirectory, 0755)
		if err != nil {
			responseFileDirectory = responseFileDirectoryFallback
		}
	}
	respFile, err := os.Create(path.Join(responseFileDirectory, responseFileName))
	if err != nil {
		responseFileDirectory = responseFileDirectoryFallback
		respFile, err = os.Create(responseFileDirectory + responseFileName)
		if err != nil {
			log.SetOutput(os.Stdout)
			log.WithFields(log.Fields{
				"responseFile": path.Join(responseFileDirectory, responseFileName),
			}).Fatal("could not create response file")
		}

	}
	mw := io.MultiWriter(os.Stdout, respFile)
	log.SetOutput(mw)
}

func postInitCommands(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitCommands(cmd.Commands())
		}
	}
}

func presetRequiredFlags(cmd *cobra.Command) {
	_ = viper.BindPFlags(cmd.Flags())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			_ = cmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}
