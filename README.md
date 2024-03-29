### How to run?

You'll need a redis server to store and retrieve the game state. This must be configured in a .env.{ENVIRONMENT} file, if no ENVIRONMENT is defined it will search for .env.dev file.

This file must contain:
```
REDIS_HOST=localhost
REDIS_PORT=6379
```

You can run a local redis service using .dev/docker-compose.yml.


Once you have it configured you can start the project using the following command:

```
make run
```

If you want to run it in release mode you can use

```
make run-release
```

**But first you shall build the project using:**

```
make build
```


### Serialized Data Structure

The gameState is serialized in binary following the next schema:

[0-64] -> Table pieces with values calculated as follows
```PIECE_TYPE (1-6) * (IF PIECE_HAS_BEEN_MOVED -> 2 | 1) * (IF PLAYER IS BLACK -> 2 | 1)```
OR
``` 0 IF SPACE IS EMPTY ```

##### What this means is values are as following: 
* 0 -> Empty board space
* 1-12 -> White player pieces, where
  * 1-6 never moved pieces
  * 7-12 moved pieces
* 13-24 -> Black player pieces, where
  * 13-18 never moved pieces
  * 19-24 moved pieces


##### 1 - 6 are pieces types
0. NO PIECE
1. PAWN
2. BISHOP
3. KNIGHT
4. ROOK
5. QUEEN
6. KING

##### PLAYER TYPES ARE THE FOLLOWING
0. WHITE PLAYER
1. BLACK PLAYER
2. UNKNOWN PLAYER (or no player)

##### How the board should be deserialized

**i.** You start by checking if the position is 0, then you set the board position to empty(PIECE_TYPE 0, PLAYER ID 2)

**ii.** If not you check if the value is > 12, if so, the piece belongs to the black player and save that the player is black, else the player is white. 

**iii.** If the player is black, substract from the value 12

**iv.** If the remaining value is > 6, then it means the piece was moved before, save as moved, if not the moved flag is false.

**v.** If the piece was moved substract from the value 6

**vi.** The remaining value should be between 1-6 and you can interpret the PIECE_TYPE from it as shown in the list before.

##### Other serialized data inside the structure

Past 64 bytes you'll find the pieces removed from the board, the format is the same, you read byte by byte untill you find a 0 byte which marks the end of the list of removed table pieces.

After that 0 byte what follows is the history of the match, this has a dynamic length, and each 2 bytes represent a start and end position on the table between 0-63. 
0 -> a1
1 -> a2
2 -> a3
.
.
.
63 -> f8

so each 2 bytes represent a full movement in UCI form like a2a4, which would be the bytes 0 and 3.
