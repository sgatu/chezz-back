package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/gookit/color"
	"github.com/redis/go-redis/v9"
	"github.com/sgatu/chezz-back/game"
	"github.com/sgatu/chezz-back/handlers"
	"github.com/sgatu/chezz-back/infrastructure/repositories"
	"github.com/sgatu/chezz-back/models"
	"github.com/sgatu/chezz-back/services"
)

func pieceToStrL(p game.PIECE_TYPE) string {
	pieceStr := pieceToStr(p)
	pieceLetter := string(pieceStr[0])
	if pieceStr == "Knight" {
		pieceLetter = "N"
	}
	return pieceLetter
}
func pieceToStr(p game.PIECE_TYPE) string {
	switch p {
	case game.PAWN:
		return "Pawn"
	case game.ROOK:
		return "Rook"
	case game.QUEEN:
		return "Queen"
	case game.KING:
		return "King"
	case game.BISHOP:
		return "Bishop"
	case game.KNIGHT:
		return "Knight"
	default:
		return "Unknown"
	}
}

func paintGame(gs *game.GameState) {
	color.Set(color.FgWhite)
	textWhite := color.FgWhite
	textBlack := color.FgBlack
	bgFirst := color.BgBlue
	bgSecond := color.BgRed
	text := ""
	lastLine := -1
	var line = 0
	for i, block := range gs.GetBoardState() {
		line = i / 8
		if lastLine != line {
			if text != "" {
				text = text + "\n"
			} else {
				text = text + "   _______________________________\n"
			}
			text = text + fmt.Sprint(line+1) + " |"
			lastLine = line
		}
		backColor := bgFirst
		if (i+line)%2 == 1 {
			backColor = bgSecond
		}
		if block != nil {
			pieceLetter := pieceToStrL(block.PieceType)
			pieceColor := textWhite
			if block.Player == game.BLACK_PLAYER {
				pieceColor = textBlack
			}

			fullColor := color.New(pieceColor, backColor, color.OpBold)
			text = text + fullColor.Render(" "+string(pieceLetter)+" ") + "|"
		} else {
			fullColor := color.New(backColor, color.OpBold)
			text = text + fullColor.Render("   ") + "|"
		}
	}
	text = text + "\n" + "   ⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻⁻\n    a   b   c   d   e   f   g   h\n"
	fmt.Print(text)
}
func clearConsole() {
	cmd := exec.Command("clear") //Linux example, its tested
	cmd.Stdout = os.Stdout
	cmd.Run()
}

type Test struct {
	Value string
}

func runConsole() {
	gameRepository := repositories.NewRedisGameRepository(redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	}))
	snowflakeNode, _ := snowflake.NewNode(1)
	gameManager := services.NewGameManagerService(gameRepository)
	thisGame := models.NewGame(snowflakeNode, 1, false)
	gameRepository.SaveGame(thisGame)
	liveGame, err := gameManager.GetLiveGameState(thisGame.Id())
	if err != nil {
		panic(err)
	}
	updatesCh := make(chan string)
	liveGame.AddObserver(updatesCh)
	defer liveGame.RemoveObserver(updatesCh)
	liveGame.StartAwaitingMoves()

	go func() {
		clearConsole()
		paintGame(thisGame.GameState())
		for {
			<-updatesCh
			clearConsole()
			paintGame(thisGame.GameState())
			player := "WHITE"
			var strMove string
			if thisGame.GameState().GetPlayerTurn() == game.BLACK_PLAYER {
				player = "BLACK"
			}
			if thisGame.GameState().GetCheckedPlayer() != game.UNKNOWN_PLAYER {
				fmt.Printf("Player %v in check\n", player)
			}
			if !thisGame.GameState().InCheckMate() {
				fmt.Printf("Next move(%s): ", player)
				fmt.Scanln(&strMove)
				liveGame.ExecuteMove(strMove)
			}
		}
	}()

	for {
		clearConsole()
		if lastErr != nil {
			fmt.Print(color.FgRed.Render(fmt.Sprintf("Last error: %s\n", lastErr)))
		}

		lastErr = nil

		player := "WHITE"
		if thisGame.GameState().GetPlayerTurn() == game.BLACK_PLAYER {
			player = "BLACK"
		}
		if thisGame.GameState().GetCheckedPlayer() != game.UNKNOWN_PLAYER {
			fmt.Printf("Player %v in check\n", player)
		}
		if !thisGame.GameState().InCheckMate() {
			fmt.Printf("Next move(%s): ", player)
			fmt.Scanln(&strMove)
		} else {
			fmt.Println("Check mate")
			os.Exit(0)
		}

		if strMove == "exit" {
			break
		}
		errState := thisGame.GameState().UpdateGameState(strMove)
		if errState != nil {
			lastErr = errState
		}
	}
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "console" {
		runConsole()
		return
	}
	router := gin.Default()
	handlers.SetupRoutes(router)
	router.Run(":8888")

}
