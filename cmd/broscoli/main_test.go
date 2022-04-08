package main

import (
	"bufio"
	"fmt"
	"github.com/hypebeast/go-osc/osc"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	SubCmdFlags = "SUB_CMD_FLAGS"
)

func TestAPI(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	address := "0.0.0.0"
	port := uint16(8765)

	fileToCreate, err := ioutil.TempFile(tempDir, "fileToCreate with spaces")
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(fileToCreate.Name())

	existingFile, err := ioutil.TempFile(tempDir, "existingFile with spaces")
	if err != nil {
		t.Fatal(err)
	}

	config := Config{
		Port:    port,
		Address: address,
		Actions: map[string]*Action{
			"/create": {
				Command:    fmt.Sprintf(`/bin/touch "%s"`, fileToCreate.Name()),
				Wait:       false,
				execParams: nil,
			},
			"/remove": {
				Command:    fmt.Sprintf(`/bin/rm "%s"`, existingFile.Name()),
				Wait:       false,
				execParams: nil,
			},
		},
	}

	err, tmpFile := writeConfig(config, tempDir)
	if err != nil {
		t.Error(err)
	}
	defer tmpFile.Close()

	ch := make(chan string)
	go runServer(tmpFile, t, ch)

	tests := []struct {
		name            string
		action          string
		file            string
		wantFileToExist bool
		wantErr         bool
	}{
		{
			name:            "creating a file",
			action:          "/action/create",
			file:            fileToCreate.Name(),
			wantFileToExist: true,
		},
		{
			name:            "removing a file",
			action:          "/action/remove",
			file:            existingFile.Name(),
			wantFileToExist: false,
		},
		{
			name:            "unknown action",
			action:          "/action/unknown",
			file:            "",
			wantFileToExist: false,
		},
	}

	select {
	case <-time.After(5 * time.Second):
		t.Errorf("Timed out")
	case <-ch:
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := osc.NewClient(address, int(port))
				message := osc.NewMessage(tt.action)
				err = client.Send(message)
				if err != nil {
					t.Errorf("Can not send message %v", err)
				}

				// Give it some time, perhaps errors show up in logs
				time.Sleep(200 * time.Millisecond)

				_, err := os.Stat(tt.file)
				exists := !os.IsNotExist(err)

				if tt.wantFileToExist != exists {
					t.Errorf("want: %v, got %v", tt.wantFileToExist, exists)
				}
			})
		}
	}
}

func runServer(config *os.File, t *testing.T, ch chan string) {
	args := []string{fmt.Sprintf("-config %s", config.Name())}

	if os.Getenv(SubCmdFlags) != "" {
		args := strings.Split(os.Getenv(SubCmdFlags), " ")
		os.Args = append([]string{os.Args[0]}, args...)
		t.Logf("Running %v\n", os.Args)
		main()
	}
	cmd := exec.Command(os.Args[0], "-test.run", t.Name())
	subEnvVar := SubCmdFlags + "=" + strings.Join(args, " ")
	cmd.Env = append(os.Environ(), subEnvVar)

	pipe, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Errorf("Can not start server")
	}
	reader := bufio.NewReader(pipe)
	line, err := reader.ReadString('\n')
	for err == nil {
		t.Log(line)
		line, err = reader.ReadString('\n')
		if strings.Contains(line, "ERR") {
			t.Error(line)
		}
		// there is no way to check if UDP server is ready
		if strings.Contains(line, "brOSColi is ready to serve") {
			ch <- "ready"
		}
	}
	cmd.Wait()
}

func writeConfig(config Config, tempDir string) (error, *os.File) {
	d, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("can not create config.yaml %v", err), nil
	}

	tmpFile, err := ioutil.TempFile(tempDir, "config-")
	if err != nil {
		return fmt.Errorf("can not write temp file %v", err), nil
	}

	_, err = tmpFile.Write(d)
	if err != nil {
		return fmt.Errorf("can not write config.yaml %v", err), nil
	}
	return err, tmpFile
}
