package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ndphu/message-handler-lib/broker"
	"github.com/ndphu/message-handler-lib/config"
	"github.com/ndphu/message-handler-lib/handler"
	"github.com/ndphu/message-handler-lib/model"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"syscall"
)

var InsideSpacesRegex = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

type Config struct {
	Managers []string `json:"managers"`
}

var conf Config

func main() {

	loadConfig()

	workerId, consumerId := config.LoadConfig()
	eventHandler, err := handler.NewEventHandler(handler.EventHandlerConfig{
		WorkerId:            workerId,
		ConsumerId:          consumerId,
		ConsumerWorkerCount: 8,
		ServiceName:         "skype-cmd-exec",
	}, func(e model.MessageEvent) {
		processMessageEvent(e)
	})

	if err != nil {
		log.Fatalf("Fail to create handler by error %v\n", err)
	}

	eventHandler.Start()

	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	log.Println("Shutdown signal received")
	eventHandler.Stop()
}

func loadConfig() {
	wd, err := os.Getwd()
	if err != nil {
		wd = ""
	}
	payload, err := ioutil.ReadFile(path.Join(wd, "config.json"))
	if err != nil {
		log.Fatalf("Fail to read config file. Error=%v\n", err)
	}
	if err := json.Unmarshal(payload, &conf); err != nil {
		log.Fatalf("Fail to parse config file. Error=%v\n", err)
	}
}

func processMessageEvent(e model.MessageEvent) {
	if e.GetFrom() == e.GetThreadId() && isManager(e.GetFrom()) {
		// direct message.
		log.Println("Executing command:", e.GetContent())
		go processCommand(e)
	}
}

func removeBlankSpaces(input string) (string) {
	final := strings.TrimSpace(input)
	return InsideSpacesRegex.ReplaceAllString(final, " ")
}

func processCommand(e model.MessageEvent) error {
	threadId := e.GetThreadId()
	reply(threadId, "Processing command...")
	command, result := execCmd(e.GetContent())
	return reply(threadId,
		"Command: "+command+"\n"+"Result:\n"+result)
}

func reply(threadId, result string) error {
	workerId := os.Getenv("WORKER_ID")
	rpc, err := broker.NewRpcClient(workerId)
	if err != nil {
		log.Printf("React: Fail to create RPC client by error %v\n", err)
		return err
	}
	request := &broker.RpcRequest{
		Method: "sendText",
		Args:   []string{threadId, wrapAsPreformatted(result)},
	}
	if err := rpc.Send(request); err != nil {
		log.Println("React: Fail to reply:", err.Error())
		return err
	}
	return nil
}

func execCmd(command string) (string, string) {
	command = removeBlankSpaces(command)
	parts := strings.Split(command, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput

	err := cmd.Start()
	if err != nil {
		return command, err.Error()
	}

	cmd.Wait()
	result := string(cmdOutput.Bytes())
	return command, result
}

func isManager(from string) bool {
	for _, m := range conf.Managers {
		if from == m {
			return true
		}
	}
	return false
}
func wrapAsPreformatted(message string) string {
	return fmt.Sprintf("<pre raw_pre=\"{code}\" raw_post=\"{code}\">%s</pre>", message)
}
