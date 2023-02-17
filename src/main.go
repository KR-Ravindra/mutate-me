package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"errors"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "k8s.io/api/core/v1"
)

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	# some issue is still existent with the serialiser, doesn't return the wanted version/apiversion/kind
)
var config *rest.Config
var clientSet *kubernetes.Clientset

type ServerParameters struct {
	port     int    // webhook server port
	certFile string // path to the x509 certificate for https
	keyFile  string // path to the x509 private key matching `CertFile`
}

var parameters ServerParameters

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)

	flag.IntVar(&parameters.port, "port", 8443, "Mutate-me server port.")
	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/ssl/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/ssl/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()
}

func main() {
	log.Println("Starting the web server on :", strconv.Itoa(parameters.port))
	// }

	// func init() {
	useKubeConfig := os.Getenv("USE_KUBECONFIG")
	kubeConfigFilePath := os.Getenv("KUBECONFIG")

	log.Info("Recieved kubeconfig flag as ", useKubeConfig)
	if len(useKubeConfig) == 0 {
		// default to service account in cluster token
		c, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		config = c
	} else {
		//load from a kube config
		var kubeconfig string

		if kubeConfigFilePath == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		} else {
			kubeconfig = kubeConfigFilePath
		}

		log.Info("kubeconfig: " + kubeconfig)

		c, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		config = c
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	clientSet = cs
	log.Info("Here is my final clientSet : ", clientSet)
	test()

	r := mux.NewRouter()
	r.HandleFunc("/", HandleRoot)
	r.HandleFunc("/mutate", HandleMutate).Methods("POST")
	log.Fatal(http.ListenAndServeTLS(":"+strconv.Itoa(parameters.port), parameters.certFile, parameters.keyFile, r))
	// log.Fatal(http.ListenAndServe(":"+strconv.Itoa(parameters.port), r))
}

func test() {
	pods, err := clientSet.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	log.Printf("There are %v pods in the cluster\n", len(pods.Items))
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	triggeredAt := time.Now()
	fmt.Fprintln(w, "hello!")

	finishedAt := time.Now()
	log.Info("Hit on / and processed in ", finishedAt.Sub(triggeredAt))
}

func HandleMutate(w http.ResponseWriter, r *http.Request) {
	triggeredAt := time.Now()
	fmt.Fprintln(w, "hello again!")
	body, err := ioutil.ReadAll(r.Body)
	err = ioutil.WriteFile("/tmp/request", body, 0644)
	if err != nil {
		panic(err.Error())
	}
	var admissionReviewReq v1beta1.AdmissionReview

	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		errors.New("malformed admission review: request is nil")
	}

	log.Printf("Type: %v \t Event: %v \t Name: %v \n",
		admissionReviewReq.Request.Kind,
		admissionReviewReq.Request.Operation,
		admissionReviewReq.Request.Name,
	)
	var pod apiv1.Pod

	err = json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod)

	if err != nil {
		fmt.Errorf("could not unmarshal pod on admission request: %v", err)
	}

	log.Printf("The raw pod request is ", string(admissionReviewReq.Request.Object.Raw))

	var patches []patchOperation
	labels := pod.ObjectMeta.Labels
	labels["example-webhook"] = "it-worked"
	log.Printf("Before Patch", patches)
	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: labels,
	})
	log.Printf("After Patch", patches)
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		fmt.Errorf("could not marshal JSON patch: %v", err)
	}
	log.Printf("Patched bytes are ", string(patchBytes))

	admissionReviewResponse := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID:     admissionReviewReq.Request.UID,
			Allowed: true,
		},
	}

	admissionReviewResponse.Response.Patch = patchBytes

	log.Printf("This is the response I am sending %v\n", &admissionReviewResponse)
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		fmt.Errorf("marshaling response: %v", err)
	}

	w.Write(bytes)

	// log.Println("Here is the body of request \n\n", body)
	// log.Println("Here is the error of request \n]n", err)
	finishedAt := time.Now()
	log.Info("Hit on /mutate and processed in ", finishedAt.Sub(triggeredAt))
}
