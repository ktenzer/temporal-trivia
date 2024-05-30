package main

import (
	"log"
	"os"

	activities "github.com/ktenzer/temporal-trivia/activities"
	"github.com/ktenzer/temporal-trivia/resources"
	workflow "github.com/ktenzer/temporal-trivia/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	c, err := client.Dial(resources.GetClientOptions("worker"))
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, os.Getenv("TEMPORAL_TASK_QUEUE"), worker.Options{})

	w.RegisterWorkflow(workflow.TriviaGameWorkflow)
	w.RegisterWorkflow(workflow.AddPlayerWorkflow)
	w.RegisterActivity(activities.GetRandomCategoryActivity)
	w.RegisterActivity(activities.TriviaQuestionChatGPT)
	w.RegisterActivity(activities.TriviaQuestionKapaAI)
	w.RegisterActivity(activities.LeaderBoardActivity)
	w.RegisterActivity(activities.ModerationActivity)
	w.RegisterActivity(activities.AddPlayerActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
