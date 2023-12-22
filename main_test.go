package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	fakeclient "k8s.io/client-go/kubernetes/fake"
)

func loadTestSecret(secretFile string) (*v1.Secret, error) {
	b, err := os.ReadFile(secretFile)
	if err != nil {
		return nil, err
	}
	var secret v1.Secret
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(b), 256)
	if err = dec.Decode(&secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

func hashDir(dirPath string) [][]byte {
	sums := make([][]byte, 1)
	files, err := filepath.Glob(fmt.Sprintf("%s/*.yaml", dirPath))
	if err != nil {
		return nil
	}
	for _, f := range files {
		f, err := os.Open(f)
		if err != nil {
			return nil
		}
		defer f.Close()

		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			return nil
		}
		sums = append(sums, h.Sum(nil))
	}
	return sums
}

func TestExport(t *testing.T) {
	// Seed the fake Kubectl client with source secret, and pass the fake into
	// SecretRetriever.
	secret, err := loadTestSecret("testdata/secret.podinfo-helm.yaml")
	if err != nil {
		t.Errorf("unable to read Secret input file")
	}
	fake := fakeclient.NewSimpleClientset(secret)
	sr := NewSecretRetriever(fake, "default")

	// Export and compare to the content of the golden folder.
	hs, err := sr.Retrieve(secret.Name)
	if err != nil {
		t.Errorf("cannot pull secret %s", secret.Name)
	}
	goldenDir := "./testdata/golden"
	outDir := t.TempDir()
	hs.Export(outDir)

	// Comapre the golden file with the output folder
	goldenHash := hashDir(goldenDir)
	outputHash := hashDir(outDir)
	if !slices.EqualFunc(goldenHash, outputHash, bytes.Equal) {
		t.Errorf("mismatch. files in the output folder %s don't match those in the golder folder %s", outDir, goldenDir)
	}
}
