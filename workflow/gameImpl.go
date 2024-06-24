package triviagame

import (
	"strings"
	"time"

	"github.com/ktenzer/temporal-trivia/resources"
	"go.temporal.io/sdk/workflow"

	_ "go.temporal.io/sdk/contrib/tools/workflowcheck/determinism"
)

func (gp *GameProgress) runGame(ctx workflow.Context, gameConfiguration *resources.GameConfiguration, getQuestions *map[int]resources.Result,
	getPlayers *map[string]resources.Player) (*map[int]resources.Result, *map[string]resources.Player) {

	logger := workflow.GetLogger(ctx)

	var questionCount int = 1
	keys := getSortedGameMap(*getQuestions)

	for _, key := range keys {
		gp.CurrentQuestion = questionCount

		// Set game progress to answer phase
		gp.Stage = "answers"

		// Async timer for amount of time to receive answers
		//timerCtx := workflow.Context(ctx)
		as := AnswerSignal{}
		questionCtx, cancelTimer := workflow.WithCancel(ctx)
		answerSelector := workflow.NewSelector(questionCtx)
		as.answerSignal(questionCtx, answerSelector)

		timer := workflow.NewTimer(questionCtx, time.Duration(gameConfiguration.AnswerTimeLimit)*time.Second)

		var timerFired bool = false
		answerSelector.AddFuture(timer, func(f workflow.Future) {
			err := f.Get(questionCtx, nil)
			if err == nil {
				logger.Info("Time limit for question has exceeded the limit of " + intToString(gameConfiguration.AnswerTimeLimit) + " seconds")
				timerFired = true
			}
		})

		// Loop through the number of players we expect to answer and break loop if question timer expires
		result := (*getQuestions)[key]
		var submissionsMap = make(map[string]resources.Submission)

		a := 0

		for a < gameConfiguration.NumberOfPlayers {
			// continue to next question if timer fires
			if timerFired {
				break
			}

			answerSelector.Select(questionCtx)

			// handle duplicate answers from same player
			var submission resources.Submission
			if as.Action == "Answer" && isPlayerValid(*getPlayers, as.Player) && !isAnswerDuplicate(submissionsMap, as.Player) && key == as.Question {
				// ensure answer is upper case
				answerUpperCase := strings.ToUpper(as.Answer)
				submission.Answer = answerUpperCase

				if result.Answer == submission.Answer {
					submission.IsCorrect = true

					if result.Winner == "" {
						result.Winner = as.Player
						submission.IsFirst = true

						if gameConfiguration.NumberOfPlayers > 1 {
							(*getPlayers)[as.Player] = resources.Player{
								Score: (*getPlayers)[as.Player].Score + 2,
							}
						} else {
							(*getPlayers)[as.Player] = resources.Player{
								Score: (*getPlayers)[as.Player].Score + 1,
							}
						}
					} else {
						(*getPlayers)[as.Player] = resources.Player{
							Score: (*getPlayers)[as.Player].Score + 1,
						}
					}
				}
				submissionsMap[as.Player] = submission
				result.Submissions = submissionsMap
			} else {
				logger.Warn("Wrong signal received", as)
			}

			(*getQuestions)[key] = result
			a++
		}
		// Cancel timer
		cancelTimer()

		// Set game progress to result phase
		gp.Stage = "result"

		// Sleep allowing time to display results
		workflow.Sleep(ctx, time.Duration(gameConfiguration.ResultTimeLimit)*time.Second)

		questionCount++
	}

	// Set game progress to result phase
	gp.Stage = "scores"

	return getQuestions, getPlayers
}
