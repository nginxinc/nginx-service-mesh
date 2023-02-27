package commands

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/support"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

// Support builds a support package containing information about the mesh.
// Building this package is best-effort. Many errors will be logged but not cause
// a crash because we want to continue on and attempt to get as much information as possible.
func Support(version string) *cobra.Command {
	output, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current working directory: %v", err)
	}

	var disableSidecarLogs bool
	cmd := &cobra.Command{
		Use:   "supportpkg",
		Short: "Create an NGINX Service Mesh support package",
		Long:  "Create an NGINX Service Mesh support package.",
	}
	cmd.Flags().StringVarP(
		&output,
		"output",
		"o",
		output,
		"output directory for supportpkg tarball",
	)
	cmd.Flags().BoolVar(
		&disableSidecarLogs,
		"disable-sidecar-logs",
		false,
		"disable the collection of sidecar logs",
	)

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := generateBundle(initK8sClient, support.NewWriter(), output, version, !disableSidecarLogs); err != nil {
			return fmt.Errorf("error generating support package: %w", err)
		}
		fmt.Println("Done.")

		return nil
	}

	return cmd
}

// generateBundle gets all of the desired support information and packages it into a tarball.
func generateBundle(k8sClient k8s.Client, writer support.FileWriter, output, version string, collectSidecarLogs bool) error {
	if _, err := verifyMeshInstall(k8sClient); err != nil {
		return fmt.Errorf("could not verify mesh: %w", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

	// print banner
	name := "nsm-supportpkg-" + time.Now().UTC().Format("20060102150405")
	filename := filepath.Join(output, fmt.Sprintf("%s.tar.gz", name))
	fmt.Printf("Saving support package to \"%s\"...\n", filename)

	// create temporary directory to hold all contents of support package
	tmpDir, err := writer.TempDir("nsm-support")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	defer removeTempDir(writer, tmpDir)

	// create top-level directory for the support package
	directory := filepath.Join(tmpDir, name)
	if err = writer.Mkdir(directory); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	go func() {
		sig := <-sigs
		removeTempDir(writer, tmpDir)
		// intentional, reserved return, 128 + signal
		os.Exit((1 << 7) | int(sig.(syscall.Signal))) //nolint // signals are signals
	}()

	// create file for holding supportpkg creation logs
	pkgLogFile, err := writer.OpenFile(filepath.Join(directory, "supportpkg-creation-logs.txt"))
	if err != nil {
		return fmt.Errorf("could not create logging file: %w", err)
	}
	defer func() {
		if closeErr := writer.Close(pkgLogFile); closeErr != nil {
			log.Fatalf("could not close temporary file: %v", closeErr)
		}
	}()

	// log to file
	log.SetOutput(pkgLogFile)

	// get mesh version and write to file
	if err := writeMeshVersion(k8sClient, writer, directory, version); err != nil {
		log.Printf("could not write NGINX Service Mesh version information: %v\n", err)
	}

	dataFetcher := support.NewDataFetcher(
		k8sClient,
		writer,
		kubeconfig,
		directory,
		collectSidecarLogs,
	)
	dataFetcher.GatherAndWriteData()

	return writer.WriteTarFile(tmpDir, filename)
}

// writeMeshVersion gets the output of nginx-meshctl version and writes to a file.
func writeMeshVersion(k8sClient k8s.Client, writer support.FileWriter, directory, version string) error {
	versionTxt := fmt.Sprintf("nginx-meshctl - v%s\n", version)
	versions, err := getComponentVersions(k8sClient.Config(), k8sClient.Namespace(), meshTimeout)
	if err != nil {
		return err
	}
	versionTxt += versions

	return writer.Write(filepath.Join(directory, "version.txt"), versionTxt)
}

func removeTempDir(writer support.FileWriter, tmpDir string) {
	log.SetOutput(os.Stderr)
	if err := writer.RemoveAll(tmpDir); err != nil {
		log.Fatalf("could not remove temporary directory '%s': %v", tmpDir, err)
	}
}
