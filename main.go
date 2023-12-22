package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	b64 "encoding/base64"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type HelmSecret struct {
	Manifest string `json:"manifest"`
}

func (h *HelmSecret) Unmsrshal(payload []byte) error {
	helmDecoded, err := b64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(helmDecoded)
	gzReader, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	d := yaml.NewYAMLOrJSONDecoder(gzReader, 256)
	d.Decode(h)
	return nil
}

func (h *HelmSecret) Export(outputDir string) error {
	docs := strings.Split(h.Manifest, "---")
	for _, d := range docs {
		if len(d) == 0 {
			continue
		}
		var obj metav1.PartialObjectMetadata
		dec := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(d), 256)
		if err := dec.Decode(&obj); err != nil {
			return err
		}
		if obj.Kind == "" {
			continue
		}
		filename := fmt.Sprintf("%s/%s.%s.%s", outputDir, strings.ToLower(obj.Kind), obj.Name, "yaml")
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			file.Write([]byte(d))
		}
		defer file.Close()
	}
	return nil
}

func defaultKubeConf() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return path.Join(homedir, ".kube", "config")
}

func defaultNamespace() string {
	return "default"
}

type Retriever interface {
	Retrieve() (v1.Secret, error)
}

type SecretRetriever struct {
	clientset kubernetes.Interface
	namespace string
}

func NewSecretRetriever(client kubernetes.Interface, namespace string) *SecretRetriever {
	sr := SecretRetriever{
		clientset: client,
		namespace: namespace,
	}
	return &sr
}

func (sr *SecretRetriever) Retrieve(secret string) (*HelmSecret, error) {
	s, err := sr.clientset.CoreV1().Secrets(sr.namespace).Get(context.TODO(), secret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var hs HelmSecret
	err = hs.Unmsrshal(s.Data["release"])
	return &hs, err
}

func main() {
	var kubeconfig string
	var namespace string
	var secret string
	var outputDir string
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeConf(), "help message for flag kubeconfig")
	flag.StringVar(&namespace, "n", defaultNamespace(), "help message for flag n")

	// Define a custom usage message, for non-flags as well
	flag.Usage = func() {
		fmt.Println("Welcome to helm-export!")
		fmt.Println("\nUsage:")
		fmt.Printf("  %s %s %s [options]\n", os.Args[0], "secret-name", "[output-dir] (if omitted, will assume the current working direcotry)")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if secret = flag.Arg(0); secret == "" {
		log.Fatalln("secret is missing")
	}
	if outputDir = flag.Arg(1); outputDir == "" {
		log.Println("Missing output directory, defaulting to the current working directory")
		outputDir, _ = os.Getwd()
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	ret := NewSecretRetriever(clientset, namespace)
	hs, _ := ret.Retrieve(secret)
	err = hs.Export(outputDir)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
