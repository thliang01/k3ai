package clusters

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"

	internal "github.com/k3ai/internal"
	color "github.com/k3ai/pkg/color"
	http "github.com/k3ai/pkg/http"
)

const (
	k3aiKube ="/.k3ai/.tools/kubectl"
	k3aiHelm = "/.k3ai/.tools/helm"
	lnxApp = "/bin/bash"
	
)
var (
	appPlugin = internal.AppPlugin{}
	kubeconfig *string
) 


func Deployment (name string, ctype string) (status bool, err error) {
		action := "install"
		appPlugin := http.InfrastructureDeployment(ctype)
		for i:=0; i < len(appPlugin.Resources); i++ {
			pluginEx := string(appPlugin.Resources[i].Path)
			if strings.Contains(pluginEx,"{{name}}") {
				pluginEx = strings.Replace(pluginEx,"{{name}}",name,-1)
			}
			
			pluginArgs := string(appPlugin.Resources[i].Args)
			if appPlugin.Resources[i].PluginType == "shell" {
				if !appPlugin.Resources[i].Wait {
					err := shell(pluginEx,pluginArgs,false,action)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					err := shell(pluginEx,pluginArgs,true, action)
					if err != nil {
						log.Fatal(err)
					}
				}

			}
		}



	status = true

	return status,nil
}

func Removal (name string, ctype string) (status bool, err error) {
	action := "removal"
	appPlugin := http.InfrastructureDeployment(ctype)
		for i:=0; i < len(appPlugin.Resources); i++ {
			pluginEx := string(appPlugin.Resources[i].Remove)
			if strings.Contains(pluginEx,"{{name}}") {
				pluginEx = strings.Replace(pluginEx,"{{name}}",name,-1)
			}
			pluginArgs := string(appPlugin.Resources[i].Args)
			if appPlugin.Resources[i].PluginType == "shell"  {
				if appPlugin.Resources[i].Wait {
					err := shell(pluginEx,pluginArgs,false, action)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					err := shell(pluginEx,pluginArgs,true, action)
					if err != nil {
						log.Fatal(err)
					}
				}

			}
	}

status = true

return status,nil
}




func WaitForDeployment(clientset *kubernetes.Clientset) {
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		
}

func Client (name string,ctype string) (clientset  *kubernetes.Clientset, kubeStr []byte) { 
	var cPath string
	if ctype == "k3s" {
		cPath ="/etc/rancher/k3s/k3s.yaml"
	} else {
		cPath = homedir.HomeDir() + "/.kube/config"
	}
	if home := homedir.HomeDir(); home != "" {	
		out,_ := os.Create(homedir.HomeDir() + "/.k3ai/" + name +".config")
		in,_ := os.Open(cPath)
	
		
		_, err := io.Copy(out,in)
		if err != nil {
			log.Print(err)
		}
		out.Close()
		

		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".k3ai","johnny_cool.config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	 
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset, kubeStr
}

func shell(pluginEx string, pluginArgs string, outPrint bool, action string) error {
		home,_ := os.UserHomeDir()
		shellPath := home + "/.k3ai"
		if pluginEx == "post" {
			pluginEx = ""
			cmd := exec.Command("/bin/bash","-c",pluginArgs)
			cmd.Dir = shellPath
			cmd.Output()


		}
		if action == "install" {
		color.Done()
		fmt.Println(" 🚀 Starting installation...")
		fmt.Println(" ")
		} else if action == "removal" {
			color.Done()
			fmt.Println(" 🚀 Removing installation...")
			fmt.Println(" ")
		}
		cmd := exec.Command("/bin/bash","-c",pluginEx,pluginArgs)
		cmd.Dir = shellPath
		r, _ := cmd.StdoutPipe()
		cmd.Stderr = cmd.Stdout
		done := make(chan struct{})

		scanner := bufio.NewScanner(r)
		go func() {
			// Read line by line and process it
			msg := "⏳	Working..."
			fmt.Printf("\r %v", msg)
			fmt.Println(" ")
			for scanner.Scan() {
				line := scanner.Text()
				color.Disable()
				if outPrint {
					fmt.Println(" 🚀 " + line)
				}
				
			}
			done <- struct{}{}
		}()
		// Start the command and check for errors
		err := cmd.Start()
		if err != nil {
			log.Println("Something went wrong... did you check all the prerequisites to run this plugin? If so try to re-run the k3ai command...")	
		}
		<-done
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
		return err
}