package game

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/sgatu/chezz-back/errors"
)

type (
	PLAYER     int
	PIECE_TYPE int
)

const (
	WHITE_PLAYER PLAYER = iota
	BLACK_PLAYER
	UNKNOWN_PLAYER
)

const (
	UNKNOWN_PIECE PIECE_TYPE = iota
	PAWN
	BISHOP
	KNIGHT
	ROOK
	QUEEN
	KING
)

type Piece struct {
	PieceType    PIECE_TYPE
	Player       PLAYER
	HasBeenMoved bool
}
type DirectionVector struct {
	x int
	y int
}
type MoveResult struct {
	Move          string
	CheckedPlayer PLAYER
	CheckMate     bool
}

func newPiece(_type PIECE_TYPE, player PLAYER, hasBeenMoved bool) *Piece {
	return &Piece{
		PieceType:    _type,
		Player:       player,
		HasBeenMoved: hasBeenMoved,
	}
}

type ActionType int

const (
	MoveAction ActionType = iota
	KillAction
)

type GameState struct {
	table         [64]*Piece
	moves         []string
	outTable      []Piece
	playerTurn    PLAYER
	checkedPlayer PLAYER
	checkMate     bool
}

type Action struct {
	uci       string
	posStart  int
	posEnd    int
	promotion PIECE_TYPE
	who       PLAYER
}

func queenDirections() []DirectionVector {
	return []DirectionVector{
		{-1, 0},
		{-1, -1},
		{0, -1},
		{0, 1},
		{1, 1},
		{1, 0},
		{1, -1},
		{-1, 1},
	}
}

func rookDirections() []DirectionVector {
	return []DirectionVector{
		{-1, 0},
		{0, -1},
		{0, 1},
		{1, 0},
	}
}

func bishopDirections() []DirectionVector {
	return []DirectionVector{
		{-1, -1},
		{1, 1},
		{1, -1},
		{-1, 1},
	}
}

var regexpUCI = regexp.MustCompile(`^([a-h][1-8])([a-h][1-8])([nbrqNBRQ]?)$`)

func coordsToPos(letter rune, pos int) (int, error) {
	p := (pos-1)*8 + strings.IndexRune("abcdefgh", letter)
	if p < 0 || p > 63 {
		return -1, fmt.Errorf("invalid coords")
	}
	return p, nil
}

func posToCoords(pos int) (rune, int, error) {
	if pos < 0 || pos > 63 {
		return ' ', 0, fmt.Errorf("invalid pos")
	}
	col := rune("abcdefgh"[pos%8])
	row := (pos / 8) + 1
	return col, row, nil
}

func getPieceFromQualifier(r byte) PIECE_TYPE {
	switch r {
	case 'B', 'b':
		return BISHOP
	case 'K', 'k':
		return KING
	case 'N', 'n':
		return KNIGHT
	case 'Q', 'q':
		return QUEEN
	case 'R', 'r':
		return ROOK
	}
	return UNKNOWN_PIECE
}

func (gs *GameState) checkIfCheckMate() bool {
	beforeState := gs.table
	for i := range gs.table {
		if gs.table[i] != nil && gs.table[i].Player == gs.playerTurn {
			moves, _ := gs.getAllAllowedMovements(i, gs.playerTurn)
			for _, mv := range moves {
				gs.table[mv] = gs.table[i]
				gs.table[i] = nil
				whiteCheck, blackCheck := gs.checkIfCheck()
				gs.table = beforeState
				if gs.playerTurn == WHITE_PLAYER && !whiteCheck {
					return false
				}
				if gs.playerTurn == BLACK_PLAYER && !blackCheck {
					return false
				}

			}
		}
	}
	return true
}

func (gs *GameState) getPawnMovements(pos int, who PLAYER) []int {
	directionMultiplier := 1
	expectedEatColor := WHITE_PLAYER
	if who == WHITE_PLAYER {
		directionMultiplier = -1
		expectedEatColor = BLACK_PLAYER
	}

	allowedMovePositions := []int{}
	if gs.table[pos+(8*directionMultiplier)] == nil {
		allowedMovePositions = append(allowedMovePositions, pos+(8*directionMultiplier))
	}
	if len(allowedMovePositions) != 0 &&
		!gs.table[pos].HasBeenMoved &&
		gs.table[allowedMovePositions[0]] == nil &&
		gs.table[pos+(16*directionMultiplier)] == nil {
		allowedMovePositions = append(allowedMovePositions, pos+(16*directionMultiplier))
	}
	rightPos := pos + (7 * directionMultiplier)
	leftPos := pos + (9 * directionMultiplier)

	columnRight := rightPos % 8
	columnLeft := leftPos % 8
	currentColumn := pos % 8
	/* check for eating movements
	 * first: check if column left or right is 1 step away(if movement leads to jump from one side of the table to another it is invalid)
	 * second: check if the new position has a enemy piece
	 */
	if math.Abs(float64(currentColumn-columnRight)) == 1 &&
		gs.table[rightPos] != nil &&
		gs.table[rightPos].Player == expectedEatColor {
		allowedMovePositions = append(allowedMovePositions, rightPos)
	}
	if math.Abs(float64(currentColumn-columnLeft)) == 1 && gs.table[leftPos] != nil && gs.table[leftPos].Player == expectedEatColor {
		allowedMovePositions = append(allowedMovePositions, leftPos)
	}
	return allowedMovePositions
}

func (gs *GameState) getKingMovements(startPos int, who PLAYER) []int {
	relativePos := []int{8, -8, 7, 9, -7, -9, -1, 1}
	eatablePlayer := gs.getOppositePlayer(who)
	allowedMovePositions := []int{}

	currentColumn := startPos % 8
	for _, pos := range relativePos {
		newPos := startPos + pos
		newPosColumn := newPos % 8
		if newPos >= 0 &&
			newPos < 64 &&
			math.Abs(float64(newPosColumn)-float64(currentColumn)) < 2 &&
			(gs.table[newPos] == nil || gs.table[newPos].Player == eatablePlayer) {
			allowedMovePositions = append(allowedMovePositions, newPos)
		}
	}
	return allowedMovePositions
}

func (gs *GameState) getAllAllowedMovements(pos int, who PLAYER) ([]int, error) {
	if pos < 0 || pos > 63 {
		return []int{}, fmt.Errorf("invalid position")
	}
	var allowedMovePositions []int
	switch gs.table[pos].PieceType {
	case PAWN:
		allowedMovePositions = gs.getPawnMovements(pos, who)
	case KING:
		allowedMovePositions = gs.getKingMovements(pos, who)
	case QUEEN:
		allowedMovePositions = gs.getQueenMovements(pos, who)
	case BISHOP:
		allowedMovePositions = gs.getBishopMovements(pos, who)
	case ROOK:
		allowedMovePositions = gs.getRookMovements(pos, who)
	case KNIGHT:
		allowedMovePositions = gs.getKnightMovements(pos, who)
	default:
		return []int{}, fmt.Errorf("invalid piece type")
	}
	return allowedMovePositions, nil
}

func (gs *GameState) getOppositePlayer(player PLAYER) PLAYER {
	/**
	below equals to:
	if gs.playerTurn == WHITE_PLAYER {
		gs.playerTurn = BLACK_PLAYER
	} else {
		gs.playerTurn = WHITE_PLAYER
	}
	*/
	return WHITE_PLAYER ^ BLACK_PLAYER ^ player
}

func (gs *GameState) getQueenMovements(pos int, who PLAYER) []int {
	return gs.getContinuousMovingPieceMovements(pos, who, queenDirections())
}

func (gs *GameState) getRookMovements(pos int, who PLAYER) []int {
	return gs.getContinuousMovingPieceMovements(pos, who, rookDirections())
}

func (gs *GameState) getBishopMovements(pos int, who PLAYER) []int {
	return gs.getContinuousMovingPieceMovements(pos, who, bishopDirections())
}

func (gs *GameState) getKnightMovements(pos int, who PLAYER) []int {
	diffs := []int{-17, -15, 15, 17, -10, -6, 10, 6}
	allowedMovePositions := []int{}
	for _, diff := range diffs {
		newPos := pos + diff
		// is a valid movement position and check if the positions is either empty or has an enemy piece
		if newPos >= 0 && newPos <= 63 &&
			math.Abs(float64(pos%8)-float64(newPos%8)) <= 2 &&
			(gs.table[newPos] == nil || gs.table[newPos].Player != who) {
			allowedMovePositions = append(allowedMovePositions, newPos)
			continue
		}
	}
	return allowedMovePositions
}

func (gs *GameState) getContinuousMovingPieceMovements(pos int, who PLAYER, directionMultipliers []DirectionVector) []int {
	skipDirections := 0
	// while it takes a bit more memory i think it's more optimal than recreating the slice when removing items, TBT(to be tested)
	directionsEnabled := make([]int, len(directionMultipliers))
	for i := range directionsEnabled {
		directionsEnabled[i] = 1
	}
	allowedMovePositions := []int{}
	distance := 0
	for {
		distance++
		for i := 0; i < len(directionsEnabled); i++ {
			if directionsEnabled[i] != 0 {

				newPos := pos + ((directionMultipliers[i].x * distance) + (8 * directionMultipliers[i].y * distance))
				// have we surpased board limits? or
				// are we crossing to opposite sides of the board? or
				// we are moving over another piece
				// then this is the last move in that direction
				if newPos < 0 || newPos > 63 || ((newPos%8 == 0 || newPos%8 == 7) && directionMultipliers[i].x != 0) || gs.table[newPos] != nil {
					directionsEnabled[i] = 0
					skipDirections++
				}
				// if the position is out of the board or the position is over another piece the player own, is illegal
				if newPos < 0 || newPos > 63 || (gs.table[newPos] != nil && gs.table[newPos].Player == who) {
					continue
				}
				allowedMovePositions = append(allowedMovePositions, newPos)
			}
		}
		if skipDirections == len(directionsEnabled) {
			break
		}
	}
	return allowedMovePositions
}

func (gs *GameState) processContinuousMovingPiece(action *Action, directionMultipliers []DirectionVector) error {
	allowedMovePositions := gs.getContinuousMovingPieceMovements(action.posStart, action.who, directionMultipliers)
	return gs.applyAction(action, allowedMovePositions)
}

func (gs *GameState) processBishopMovement(action *Action) error {
	return gs.processContinuousMovingPiece(action, bishopDirections())
}

func (gs *GameState) processRookMovement(action *Action) error {
	return gs.processContinuousMovingPiece(action, rookDirections())
}

func (gs *GameState) processQueenMovement(action *Action) error {
	return gs.processContinuousMovingPiece(action, queenDirections())
}

func (gs *GameState) processKnightMovement(action *Action) error {
	allowedMovePositions := gs.getKnightMovements(action.posStart, action.who)
	return gs.applyAction(action, allowedMovePositions)
}

func (gs *GameState) processPawnMovement(action *Action) error {
	allowedMovePositions := gs.getPawnMovements(action.posStart, action.who)
	if !slices.Contains(allowedMovePositions, action.posEnd) {
		return &errors.InvalidMoveError{
			Message: fmt.Sprintf("not a valid movement, allowed moves %+v", allowedMovePositions),
			ErrCode: "MOVE_NOT_ALLOWED",
		}
	}

	if (action.posEnd < 8 && action.who == BLACK_PLAYER) || (action.posEnd > 55 && action.who == WHITE_PLAYER) {
		if action.promotion == UNKNOWN_PIECE {
			return &errors.InvalidMoveError{
				Message: "move requires promotion",
				ErrCode: "MOVE_MISSING_PROMOTION",
			}
		}
		gs.table[action.posEnd] = newPiece(action.promotion, action.who, true)
		gs.table[action.posStart] = nil
		return nil
	} else {
		return gs.applyAction(action, allowedMovePositions)
	}
}

func (gs *GameState) processKingMovement(action *Action) error {
	allowedMovePositions := gs.getKingMovements(action.posStart, action.who)
	return gs.applyAction(action, allowedMovePositions)
}

func (gs *GameState) applyAction(action *Action, allowedMovePositions []int) error {
	if !slices.Contains(allowedMovePositions, action.posEnd) {
		return &errors.InvalidMoveError{
			Message: fmt.Sprintf("invalid movement, allowed %+v", allowedMovePositions),
			ErrCode: "MOVE_NOT_ALLOWED",
		}
	}
	if gs.table[action.posEnd] != nil {
		gs.outTable = append(gs.outTable, *gs.table[action.posEnd])
	}
	gs.table[action.posStart].HasBeenMoved = true
	gs.table[action.posEnd] = gs.table[action.posStart]
	gs.table[action.posStart] = nil
	return nil
}

func (gs *GameState) checkIfCheck() (bool, bool) {
	whiteKingPos := -1
	blackKingPos := -1
	for i := range gs.table {
		if gs.table[i] != nil && gs.table[i].PieceType == KING {
			if gs.table[i].Player == WHITE_PLAYER {
				whiteKingPos = i
			} else {
				blackKingPos = i
			}
		}
		if blackKingPos != -1 && whiteKingPos != -1 {
			break
		}
	}
	isWhiteChecked := false
	isBlackChecked := false
	for i := range gs.table {
		if gs.table[i] != nil {
			moves, _ := gs.getAllAllowedMovements(i, gs.table[i].Player)
			if !isWhiteChecked && gs.table[i].Player == BLACK_PLAYER && slices.Contains(moves, whiteKingPos) {
				isWhiteChecked = true
			}
			if !isBlackChecked && gs.table[i].Player == WHITE_PLAYER && slices.Contains(moves, blackKingPos) {
				isBlackChecked = true
			}
		}
		if isWhiteChecked && isBlackChecked {
			break
		}
	}
	return isWhiteChecked, isBlackChecked
}

func NewGameState() *GameState {
	table := [64]*Piece{}
	for i := 0; i < 8; i++ {
		switch i {
		case 0, 7:
			table[i] = newPiece(ROOK, BLACK_PLAYER, false)
			table[63-i] = newPiece(ROOK, WHITE_PLAYER, false)
		case 1, 6:
			table[i] = newPiece(KNIGHT, BLACK_PLAYER, false)
			table[63-i] = newPiece(KNIGHT, WHITE_PLAYER, false)
		case 2, 5:
			table[i] = newPiece(BISHOP, BLACK_PLAYER, false)
			table[63-i] = newPiece(BISHOP, WHITE_PLAYER, false)
		case 3:
			table[i] = newPiece(QUEEN, BLACK_PLAYER, false)
			table[63-i-1] = newPiece(QUEEN, WHITE_PLAYER, false)
		case 4:
			table[i] = newPiece(KING, BLACK_PLAYER, false)
			table[63-i+1] = newPiece(KING, WHITE_PLAYER, false)
		}
		table[8+i] = newPiece(PAWN, BLACK_PLAYER, false)
		table[63-8-i] = newPiece(PAWN, WHITE_PLAYER, false)
	}

	return &GameState{
		playerTurn:    WHITE_PLAYER,
		table:         table,
		outTable:      []Piece{},
		checkMate:     false,
		checkedPlayer: UNKNOWN_PLAYER,
		moves:         []string{},
	}
}

func FromSerialized(serializedData []byte) (*GameState, error) {
	playerTurn := WHITE_PLAYER
	checkedPlayer := UNKNOWN_PLAYER
	isCheckMate := false
	table := [64]*Piece{}
	outPieces := []Piece{}
	moves := []string{}
	readingMoves := false
	pieceFromByte := func(b byte) *Piece {
		player := WHITE_PLAYER
		hasBeenMoved := false
		bCopy := b
		if b > 12 {
			player = BLACK_PLAYER
			bCopy -= 12
		}

		if bCopy > 6 {
			hasBeenMoved = true
			bCopy -= 6
			if bCopy > 6 {
				return nil
			}
		}

		return &Piece{
			PieceType:    PIECE_TYPE(bCopy),
			Player:       player,
			HasBeenMoved: hasBeenMoved,
		}
	}
	bytesToMove := func(startPos byte, endPos byte) (string, error) {
		startCol, startRow, errStart := posToCoords(int(startPos))
		endCol, endRow, errEnd := posToCoords(int(endPos))
		if errStart == nil && errEnd == nil {
			return fmt.Sprintf("%c%d%c%d", startCol, startRow, endCol, endRow), nil
		}
		return "", fmt.Errorf("could not convert bytes")
	}
	// used to recover moves
	startPos := byte(255)
	endPos := byte(255)
	for i, b := range serializedData {
		if i == 0 {
			if b == 1 {
				playerTurn = BLACK_PLAYER
			}
			continue
		}
		if i == 1 {
			checkedPlayer = PLAYER(b)
			continue
		}
		if i == 2 {
			isCheckMate = b == 1
			continue
		}
		if i < 67 {
			if b != 0 {
				table[i-3] = pieceFromByte(b)
			}
		} else {
			if b == 0 && !readingMoves {
				readingMoves = true
			} else if b != 0 && !readingMoves {
				outPieces = append(outPieces, *pieceFromByte(b))
			} else {
				if startPos == 255 {
					startPos = b
				} else {
					endPos = b
					move, errMove := bytesToMove(startPos, endPos)
					if errMove == nil {
						moves = append(moves, move)
					}
					startPos = 255
				}
			}
		}
	}
	return &GameState{
		playerTurn:    playerTurn,
		table:         table,
		outTable:      outPieces,
		moves:         moves,
		checkedPlayer: checkedPlayer,
		checkMate:     isCheckMate,
	}, nil
}

func (gs *GameState) Serialize() ([]byte, error) {
	pieceToByte := func(p *Piece) byte {
		if p == nil {
			return 0
		}
		typeB := p.PieceType
		if p.Player == BLACK_PLAYER {
			typeB += 12
		}
		if p.HasBeenMoved {
			typeB += 6
		}
		return byte(typeB)
	}
	returnBytes := make([]byte, 0, 67)
	pieceBytes := make([]byte, 0, 64)
	for _, p := range gs.table {
		pieceBytes = append(pieceBytes, pieceToByte(p))
	}

	returnBytes = append(returnBytes, byte(gs.playerTurn))
	returnBytes = append(returnBytes, byte(gs.checkedPlayer))
	if gs.checkMate {
		returnBytes = append(returnBytes, byte(1))
	} else {
		returnBytes = append(returnBytes, byte(0))
	}
	returnBytes = append(returnBytes, pieceBytes...)
	for _, outPiece := range gs.outTable {
		returnBytes = append(returnBytes, pieceToByte(&outPiece))
	}
	returnBytes = append(returnBytes, 0)
	for _, move := range gs.moves {
		start, errStart := coordsToPos(rune(move[0]), int(move[1]-'0'))
		end, errEnd := coordsToPos(rune(move[2]), int(move[3]-'0'))
		if errStart == nil && errEnd == nil {
			returnBytes = append(returnBytes, byte(start), byte(end))
		}
	}
	return returnBytes, nil
}

func (gs *GameState) uci2Action(action string) (*Action, *errors.UnparseableMoveError) {
	matches := regexpUCI.FindStringSubmatch(action)
	if matches == nil {
		return nil, &errors.UnparseableMoveError{}
	}
	startPos, _ := coordsToPos(rune(matches[1][0]), int(matches[1][1]-'0'))
	endPos, _ := coordsToPos(rune(matches[2][0]), int(matches[2][1]-'0'))
	promotion := UNKNOWN_PIECE
	if len(matches[3]) > 0 {
		promotion = getPieceFromQualifier(matches[3][0])
		if promotion == KING || promotion == PAWN {
			promotion = UNKNOWN_PIECE
		}
	}
	return &Action{
		posStart:  startPos,
		posEnd:    endPos,
		promotion: promotion,
		who:       gs.playerTurn,
		uci:       action,
	}, nil
}

// PUBLIC METHODS
func (gs *GameState) GetPlayerTurn() PLAYER {
	return gs.playerTurn
}

func (gs *GameState) GetBoardState() [64]*Piece {
	return gs.table
}

func (gs *GameState) InCheckMate() bool {
	return gs.checkMate
}

func (gs *GameState) GetCheckedPlayer() PLAYER {
	return gs.checkedPlayer
}

func (gs *GameState) UpdateGameState(uciAction string) (*MoveResult, error) {
	action, err := gs.uci2Action(uciAction)
	if err != nil {
		return nil, err
	}
	if gs.checkMate {
		return nil, &errors.InvalidMoveError{
			Message: "Game in checkmate",
			ErrCode: "CHECKMATE",
		}
	}
	// control make no sense, action.who is populated from gs.playerTurn, this will never fail
	// if action.who != gs.playerTurn {
	// 	return &InvalidMoveError{
	// 		message: "Not your turn",
	// 		code:    "INVALID_TURN_MOVE",
	// 	}
	// }
	if gs.table[action.posStart] == nil || gs.table[action.posStart].Player != action.who {
		return nil, &errors.InvalidMoveError{
			Message: "No piece selected or piece not owned",
			ErrCode: "INVALID_PIECE_SELECTED",
		}
	}
	if gs.table[action.posEnd] != nil && gs.table[action.posEnd].Player == action.who {
		return nil, &errors.InvalidMoveError{
			Message: "Move position invalid, already occupied by another piece",
			ErrCode: "INVALID_POSITION",
		}
	}
	if action.posStart == action.posEnd {
		return nil, &errors.InvalidMoveError{
			Message: "No move made",
			ErrCode: "NO_MOVE",
		}
	}
	beforeState := gs.table
	var processErr error
	switch gs.table[action.posStart].PieceType {
	case PAWN:
		processErr = gs.processPawnMovement(action)
	case BISHOP:
		processErr = gs.processBishopMovement(action)
	case KING:
		processErr = gs.processKingMovement(action)
	case QUEEN:
		processErr = gs.processQueenMovement(action)
	case ROOK:
		processErr = gs.processRookMovement(action)
	case KNIGHT:
		processErr = gs.processKnightMovement(action)
	}
	if processErr != nil {
		return nil, processErr
	}
	whiteCheck, blackCheck := gs.checkIfCheck()
	if (whiteCheck && gs.playerTurn == WHITE_PLAYER) || (blackCheck && gs.playerTurn == BLACK_PLAYER) {
		gs.table = beforeState
		return nil, &errors.InvalidMoveError{
			Message: "Move should not result in check",
			ErrCode: "MOVE_IN_CHECK",
		}
	}
	gs.moves = append(gs.moves, action.uci)
	gs.checkedPlayer = UNKNOWN_PLAYER
	if whiteCheck {
		gs.checkedPlayer = WHITE_PLAYER
	} else if blackCheck {
		gs.checkedPlayer = BLACK_PLAYER
	}
	gs.playerTurn = gs.getOppositePlayer(gs.playerTurn)
	if (gs.checkedPlayer != UNKNOWN_PLAYER) && gs.checkIfCheckMate() {
		gs.checkMate = true
	}
	return &MoveResult{
		Move:          uciAction,
		CheckedPlayer: gs.checkedPlayer,
		CheckMate:     gs.checkMate,
	}, nil
}
