package game

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/sgatu/chezz-back/errors"
)

const PROTOCOL_VERSION = 1

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
	Move             string
	EnPassantCapture string
	CheckedPlayer    PLAYER
	MateStatus       GameStateStatus
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

type CastleRights struct {
	whiteQueenSide bool
	whiteKingSide  bool
	blackKingSide  bool
	blackQueenSide bool
}

func (cr CastleRights) Serialize() byte {
	castleRights := byte(0)
	if cr.whiteQueenSide {
		castleRights |= 1
	}
	if cr.whiteKingSide {
		castleRights |= 2
	}
	if cr.blackQueenSide {
		castleRights |= 4
	}
	if cr.blackKingSide {
		castleRights |= 8
	}
	return castleRights
}

func castleRightsFromByte(b byte) CastleRights {
	cr := CastleRights{}
	if b&1 == 1 {
		cr.whiteQueenSide = true
	}
	if b&2 == 2 {
		cr.whiteKingSide = true
	}
	if b&4 == 4 {
		cr.blackQueenSide = true
	}
	if b&8 == 8 {
		cr.blackKingSide = true
	}
	return cr
}

type GameStateStatus int

const (
	STATUS_PLAYING GameStateStatus = iota
	STATUS_CHECKMATE
	STATUS_STALEMATE
)

type GameState struct {
	major_version    int
	table            [64]*Piece
	moves            []string
	outTable         []Piece
	playerTurn       PLAYER
	checkedPlayer    PLAYER
	gameStatus       GameStateStatus
	lastMoveIsAPJump bool
	castleRights     CastleRights
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

var regexpUCI = regexp.MustCompile(`^([a-h][1-8])([a-h][1-8])([nbrqNBRQ]?|(e\.p)?)$`)

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

func (gs *GameState) checkIfMate() GameStateStatus {
	newStatus := gs.gameStatus
	beforeState := gs.table
	for i := range gs.table {
		if newStatus != STATUS_PLAYING {
			break
		}
		if gs.table[i] != nil && gs.table[i].Player == gs.playerTurn {
			moves, _ := gs.getAllAllowedMovements(i, gs.playerTurn)
			for _, mv := range moves {
				gs.table[mv] = gs.table[i]
				gs.table[i] = nil
				whiteCheck, blackCheck := gs.checkIfCheck()
				gs.table = beforeState
				if gs.playerTurn == WHITE_PLAYER && !whiteCheck {
					return STATUS_PLAYING
				}
				if gs.playerTurn == BLACK_PLAYER && !blackCheck {
					return STATUS_PLAYING
				}

			}
		}
	}
	if gs.checkedPlayer != UNKNOWN_PLAYER {
		return STATUS_CHECKMATE
	}
	return STATUS_STALEMATE
}

func posInRange(pos int) bool {
	return pos >= 0 && pos < 64
}

func (gs *GameState) isEnPassantMovement(startPos int, endPos int, who PLAYER) bool {
	if !gs.lastMoveIsAPJump || len(gs.moves) == 0 {
		return false
	}
	directionMultiplier := getDirection(startPos, endPos)
	enPassantRightPos := startPos + (-1 * directionMultiplier)
	enPassantLeftPos := startPos + (1 * directionMultiplier)
	checkPos := enPassantLeftPos
	moveDiff := math.Abs(float64(endPos - startPos))
	if moveDiff < 8 {
		checkPos = enPassantRightPos
	}
	lastAction, _ := gs.uci2Action(gs.moves[len(gs.moves)-1])
	if lastAction.posEnd != checkPos {
		return false
	}
	return (gs.table[checkPos] != nil && gs.table[checkPos].PieceType == PAWN && gs.table[checkPos].Player == gs.getOppositePlayer(who))
}

func (gs *GameState) getPawnMovements(pos int, who PLAYER) []int {
	directionMultiplier := 1
	expectedEatColor := BLACK_PLAYER
	if who == BLACK_PLAYER {
		directionMultiplier = -1
		expectedEatColor = WHITE_PLAYER
	}
	allowedMovePositions := []int{}
	forwardPos := pos + (8 * directionMultiplier)
	if posInRange(forwardPos) && gs.table[forwardPos] == nil {
		allowedMovePositions = append(allowedMovePositions, pos+(8*directionMultiplier))
	}
	forwardJumpPos := pos + (16 * directionMultiplier)
	if posInRange(forwardJumpPos) &&
		len(allowedMovePositions) != 0 &&
		!gs.table[pos].HasBeenMoved &&
		gs.table[allowedMovePositions[0]] == nil &&
		gs.table[forwardJumpPos] == nil {
		allowedMovePositions = append(allowedMovePositions, pos+(16*directionMultiplier))
	}
	rightPos := pos + (7 * directionMultiplier)
	leftPos := pos + (9 * directionMultiplier)

	if !posInRange(rightPos) && !posInRange(leftPos) {
		return allowedMovePositions
	}
	columnRight := rightPos % 8
	columnLeft := leftPos % 8
	currentColumn := pos % 8
	/* check for eating movements
		 * first: check if column left or right is 1 step away(if movement leads to jump from one side of the table to another it is invalid)
		 * second: check if the new position has a enemy piece if so, it's allowed to capture it
	   *  else if there is an enemy pawn next to my pawn that hast just jumped, we are allowed to capture it too
	   * same check for both columns
	*/
	if posInRange(rightPos) && math.Abs(float64(currentColumn-columnRight)) == 1 {
		if gs.table[rightPos] != nil && gs.table[rightPos].Player == expectedEatColor {
			allowedMovePositions = append(allowedMovePositions, rightPos)
		} else if gs.isEnPassantMovement(pos, rightPos, who) {
			allowedMovePositions = append(allowedMovePositions, rightPos)
		}
	}
	if posInRange(leftPos) && math.Abs(float64(currentColumn-columnLeft)) == 1 {
		if gs.table[leftPos] != nil && gs.table[leftPos].Player == expectedEatColor {
			allowedMovePositions = append(allowedMovePositions, leftPos)
		} else if gs.isEnPassantMovement(pos, leftPos, who) {
			allowedMovePositions = append(allowedMovePositions, leftPos)
		}
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
	allowedMovePositions = append(allowedMovePositions, gs.getKingCastleRightsMovements(who)...)
	return allowedMovePositions
}

func (gs *GameState) getKingCastleRightsMovements(who PLAYER) []int {
	kingSide := gs.castleRights.whiteKingSide
	queenSide := gs.castleRights.whiteQueenSide
	kingFree := []int{5, 6}
	kingMoveEndPos := 6
	queenFree := []int{1, 2, 3}
	queenMoveEndPos := 2
	allowedMovePositions := []int{}
	if who == BLACK_PLAYER {
		kingSide = gs.castleRights.blackKingSide
		queenSide = gs.castleRights.blackQueenSide
		kingFree = []int{62, 61}
		kingMoveEndPos = 62
		queenFree = []int{59, 58, 57}
		queenMoveEndPos = 58
	}
	if kingSide && gs.table[kingFree[0]] == nil && gs.table[kingFree[1]] == nil {
		allowedMovePositions = append(allowedMovePositions, kingMoveEndPos)
	}
	if queenSide &&
		gs.table[queenFree[0]] == nil &&
		gs.table[queenFree[1]] == nil &&
		gs.table[queenFree[2]] == nil {
		allowedMovePositions = append(allowedMovePositions, queenMoveEndPos)
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

	if (action.posEnd > 55 && action.who == WHITE_PLAYER) || (action.posEnd < 8 && action.who == BLACK_PLAYER) {
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

/**
 * returns true if is a castling movement. the other two values are first the rook position and the second the rook end position
 */
func (gs *GameState) isCastlingMovement(action *Action) (bool, int, int) {
	dist := float64(action.posEnd - action.posStart)
	absdist := math.Abs(dist)
	direction := 1
	if dist < 0 {
		direction = -1
	}
	isCastling := gs.table[action.posStart].PieceType == KING &&
		absdist == 2 &&
		!gs.table[action.posStart].HasBeenMoved

	if isCastling {
		rookEnd := action.posStart + direction
		rookStart := action.posStart + (int(absdist)+1)*direction
		if direction < 0 {
			rookStart -= 1
		}
		return isCastling, rookStart, rookEnd
	}
	return false, 0, 0
}

func (gs *GameState) updateCastleRights(action *Action, moving *Piece, eaten *Piece) {
	if moving == nil {
		return
	}
	switch moving.PieceType {
	case ROOK:
		switch action.posStart {
		case 0:
			gs.castleRights.whiteQueenSide = false
		case 7:
			gs.castleRights.whiteKingSide = false
		case 56:
			gs.castleRights.blackQueenSide = false
		case 63:
			gs.castleRights.blackKingSide = false
		}
	case KING:
		if action.who == WHITE_PLAYER {
			gs.castleRights.whiteKingSide = false
			gs.castleRights.whiteQueenSide = false
		} else {
			gs.castleRights.blackKingSide = false
			gs.castleRights.blackQueenSide = false
		}
	}
	if eaten != nil && eaten.PieceType == ROOK {
		switch action.posEnd {
		case 0:
			gs.castleRights.whiteQueenSide = false
		case 7:
			gs.castleRights.whiteKingSide = false
		case 56:
			gs.castleRights.blackQueenSide = false
		case 63:
			gs.castleRights.blackKingSide = false
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getDirection(startPos int, endPos int) int {
	sign := boolToInt(math.Signbit(float64(startPos - endPos)))
	return int(2*sign - 1)
}

func (gs *GameState) applyAction(action *Action, allowedMovePositions []int) error {
	if !slices.Contains(allowedMovePositions, action.posEnd) {
		return &errors.InvalidMoveError{
			Message: fmt.Sprintf("invalid movement, allowed %+v", allowedMovePositions),
			ErrCode: "MOVE_NOT_ALLOWED",
		}
	}
	moving := gs.table[action.posStart]
	eaten := gs.table[action.posEnd]
	if gs.table[action.posEnd] != nil {
		gs.outTable = append(gs.outTable, *gs.table[action.posEnd])
	}
	if isCastling, rookStart, rookEnd := gs.isCastlingMovement(action); isCastling {
		gs.table[rookEnd] = gs.table[rookStart]
		gs.table[rookEnd].HasBeenMoved = true
		gs.table[rookStart] = nil
	}
	if gs.isEnPassantMovement(action.posStart, action.posEnd, action.who) {
		direction := getDirection(action.posStart, action.posEnd)
		gs.outTable = append(gs.outTable, *gs.table[action.posEnd-(direction*8)])
		gs.table[action.posEnd-(direction*8)] = nil
	}
	gs.table[action.posEnd] = gs.table[action.posStart]
	gs.table[action.posStart] = nil
	gs.table[action.posEnd].HasBeenMoved = true
	gs.updateCastleRights(action, moving, eaten)
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
			table[i] = newPiece(ROOK, WHITE_PLAYER, false)
			table[63-i] = newPiece(ROOK, BLACK_PLAYER, false)
		case 1, 6:
			table[i] = newPiece(KNIGHT, WHITE_PLAYER, false)
			table[63-i] = newPiece(KNIGHT, BLACK_PLAYER, false)
		case 2, 5:
			table[i] = newPiece(BISHOP, WHITE_PLAYER, false)
			table[63-i] = newPiece(BISHOP, BLACK_PLAYER, false)
		case 3:
			table[i] = newPiece(QUEEN, WHITE_PLAYER, false)
			table[63-i-1] = newPiece(QUEEN, BLACK_PLAYER, false)
		case 4:
			table[i] = newPiece(KING, WHITE_PLAYER, false)
			table[63-i+1] = newPiece(KING, BLACK_PLAYER, false)
		}
		table[8+i] = newPiece(PAWN, WHITE_PLAYER, false)
		table[63-8-i] = newPiece(PAWN, BLACK_PLAYER, false)
	}

	return &GameState{
		major_version:    PROTOCOL_VERSION,
		playerTurn:       WHITE_PLAYER,
		table:            table,
		outTable:         []Piece{},
		gameStatus:       STATUS_PLAYING,
		checkedPlayer:    UNKNOWN_PLAYER,
		moves:            []string{},
		lastMoveIsAPJump: false,
		castleRights: CastleRights{
			whiteQueenSide: true,
			blackQueenSide: true,
			whiteKingSide:  true,
			blackKingSide:  true,
		},
	}
}

func FromSerialized(serializedData []byte) (*GameState, error) {
	playerTurn := WHITE_PLAYER
	checkedPlayer := UNKNOWN_PLAYER
	gameStatus := STATUS_PLAYING
	table := [64]*Piece{}
	outPieces := []Piece{}
	moves := []string{}
	readingMoves := false
	pieceFromByte := func(b byte) *Piece {
		player := WHITE_PLAYER
		if b&8 == 8 {
			player = BLACK_PLAYER
		}
		hasBeenMoved := (b & 16) == 16
		b = (b & 7)
		return &Piece{
			PieceType:    PIECE_TYPE(b),
			Player:       player,
			HasBeenMoved: hasBeenMoved,
		}
	}
	tagFromByte := func(b byte) string {
		switch b {
		case 1:
			return "Q"
		case 2:
			return "N"
		case 3:
			return "B"
		case 4:
			return "R"
		case 5:
			return "e.p"
		default:
			return ""
		}
	}
	hasTag := func(b byte) (bool, byte) {
		hasTag := false
		if b&128 == 128 {
			hasTag = true
			b &= 127
		}
		return hasTag, b
	}
	bytesToMove := func(movement [3]byte) (string, error) {
		startCol, startRow, errStart := posToCoords(int(movement[0]))
		endCol, endRow, errEnd := posToCoords(int(movement[1]))

		tag := tagFromByte(movement[2])
		if errStart == nil && errEnd == nil {
			return fmt.Sprintf("%c%d%c%d%s", startCol, startRow, endCol, endRow, tag), nil
		}
		return "", fmt.Errorf("could not convert bytes")
	}
	var castleRights CastleRights
	lastMoveIsAPJump := false
	// used to recover moves
	historyMovement := [3]byte{}
	idx := 0
	major_version := 0
	for i, b := range serializedData {
		if i == 0 {
			major_version = int(b >> 3)
			lastMoveIsAPJump = b&4 != 0
			playerTurn = PLAYER(b & 3)
			continue
		}
		if i == 1 {
			checkedPlayer = PLAYER(b)
			continue
		}
		if i == 2 {
			gameStatus = GameStateStatus(b)
			continue
		}
		if i == 3 {
			castleRights = castleRightsFromByte(b)
			continue
		}
		if i < 68 {
			if b != 0 {
				table[i-4] = pieceFromByte(b)
			}
		} else {
			if b == 0 && !readingMoves {
				readingMoves = true
			} else if b != 0 && !readingMoves {
				outPieces = append(outPieces, *pieceFromByte(b))
			} else {
				if idx == 1 {
					if hasTag, b := hasTag(b); hasTag {
						historyMovement[idx] = b
						idx++
						continue
					}
				}
				historyMovement[idx] = b
				idx++
				if idx >= 2 {
					move, errMove := bytesToMove(historyMovement)
					historyMovement[2] = 0
					idx = 0
					if errMove == nil {
						moves = append(moves, move)
					}
				}
			}
		}
	}
	return &GameState{
		major_version:    major_version,
		playerTurn:       playerTurn,
		table:            table,
		outTable:         outPieces,
		moves:            moves,
		checkedPlayer:    checkedPlayer,
		gameStatus:       gameStatus,
		castleRights:     castleRights,
		lastMoveIsAPJump: lastMoveIsAPJump,
	}, nil
}

func (gs *GameState) Serialize() ([]byte, error) {
	pieceToByte := func(p *Piece) byte {
		if p == nil {
			return 0
		}
		typeB := p.PieceType
		if p.Player == BLACK_PLAYER {
			typeB |= 8
		}
		if p.HasBeenMoved {
			typeB |= 16
		}
		return byte(typeB)
	}
	promotionCharToByte := func(c rune) byte {
		switch c {
		case 'Q':
			return 1
		case 'N':
			return 2
		case 'B':
			return 3
		case 'R':
			return 4
		default:
			return 0
		}
	}
	returnBytes := make([]byte, 0, 68)
	pieceBytes := make([]byte, 0, 64)
	for _, p := range gs.table {
		pieceBytes = append(pieceBytes, pieceToByte(p))
	}
	header := byte(gs.major_version<<3) | byte(gs.playerTurn)
	if gs.lastMoveIsAPJump {
		header |= 4
	}
	returnBytes = append(returnBytes, header)
	returnBytes = append(returnBytes, byte(gs.checkedPlayer))
	returnBytes = append(returnBytes, byte(gs.gameStatus))
	returnBytes = append(returnBytes, gs.castleRights.Serialize())
	returnBytes = append(returnBytes, pieceBytes...)
	for _, outPiece := range gs.outTable {
		returnBytes = append(returnBytes, pieceToByte(&outPiece))
	}
	returnBytes = append(returnBytes, 0)
	for _, move := range gs.moves {
		start, errStart := coordsToPos(rune(move[0]), int(move[1]-'0'))
		end, errEnd := coordsToPos(rune(move[2]), int(move[3]-'0'))
		tag := byte(0)
		if len(move) == 5 {
			tag = promotionCharToByte(rune(move[4]))
			end |= 128
		}
		if len(move) == 7 && strings.HasSuffix(move, "e.p") {
			tag = 5
			end |= 128
		}
		if errStart == nil && errEnd == nil {
			returnBytes = append(returnBytes, byte(start), byte(end))
		}
		if tag > 0 {
			returnBytes = append(returnBytes, tag)
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
	return gs.gameStatus == STATUS_CHECKMATE
}

func (gs *GameState) InStalemate() bool {
	return gs.gameStatus == STATUS_STALEMATE
}

func (gs *GameState) GetCheckedPlayer() PLAYER {
	return gs.checkedPlayer
}

func (gs *GameState) UpdateGameState(uciAction string) (*MoveResult, error) {
	action, err := gs.uci2Action(uciAction)
	if err != nil {
		return nil, err
	}
	if gs.gameStatus == STATUS_CHECKMATE {
		return nil, &errors.InvalidMoveError{
			Message: "Game in checkmate",
			ErrCode: "CHECKMATE",
		}
	}
	if gs.gameStatus == STATUS_STALEMATE {
		return nil, &errors.InvalidMoveError{
			Message: "Game in stalemate",
			ErrCode: "STALEMATE",
		}
	}

	if gs.table[action.posStart] == nil || gs.table[action.posStart].Player != action.who {
		return nil, &errors.InvalidMoveError{
			Message: "No piece selected or piece not owned",
			ErrCode: "INVALID_PIECE_SELECTED",
		}
	}
	// check if the end position is not already used by another piece
	if gs.table[action.posEnd] != nil &&
		gs.table[action.posEnd].Player == action.who {
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

	isPawnJump := int32(math.Abs(float64(action.posStart-action.posEnd))) == 16 &&
		!gs.table[action.posStart].HasBeenMoved &&
		gs.table[action.posStart].PieceType == PAWN
	enPassantMovement := gs.isEnPassantMovement(action.posStart, action.posEnd, action.who)
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
	uciMovement := action.uci
	if enPassantMovement {
		uciMovement += "e.p"
	}
	gs.moves = append(gs.moves, uciMovement)
	gs.checkedPlayer = UNKNOWN_PLAYER
	if whiteCheck {
		gs.checkedPlayer = WHITE_PLAYER
	} else if blackCheck {
		gs.checkedPlayer = BLACK_PLAYER
	}
	gs.playerTurn = gs.getOppositePlayer(gs.playerTurn)
	gs.gameStatus = gs.checkIfMate()
	gs.lastMoveIsAPJump = isPawnJump
	enPassantCapture := ""
	if enPassantMovement {
		direction := getDirection(action.posStart, action.posEnd)
		letter, number, _ := posToCoords(action.posEnd - (direction * 8))
		enPassantCapture = fmt.Sprintf("%c%d", letter, number)
	}
	return &MoveResult{
		Move:             uciAction,
		CheckedPlayer:    gs.checkedPlayer,
		MateStatus:       gs.gameStatus,
		EnPassantCapture: enPassantCapture,
	}, nil
}
