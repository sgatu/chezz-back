## How to run?

You'll need a redis server to store and retrieve the game state. This must be configured in a .env.{ENVIRONMENT} file, if no ENVIRONMENT is defined it will search for .env.dev file.

This file must contain (actual configuration may differ):
```
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PREFIX=chezz:
ALLOWED_DOMAIN=http://front-end.domain
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


## Serialized Data Structure

The gameState is serialized in binary following the next schema (first column are byte positions):

```
- byte at 0 -> Player turn -> 0 - WHITE PLAYER, 1 - BLACK PLAYER
- byte at 1 -> Checked player -> 0 - WHITE PLAYER, 1 - BLACK PLAYER, 2 - NO PLAYER
- byte at 2 -> Is game in checkMate -> 0 - No, 1 - Yes
- byte at 3 -> Castle rights -> Single byte with bit flags as following: 
    &1 - White Queen Side
    &2 - White King Side
    &4 - Black Queen Side
    &8 - Black King Side 
- bytes between [4-67] -> Table positions with values calculated as follows

    PIECE_TYPE (1-6) * (IF PIECE_HAS_BEEN_MOVED -> 2 | 1) * (IF PLAYER IS BLACK -> 2 | 1)
      OR
    0 IF SPACE IS Empty

- bytes [68...until we find a 0 byte] -> Captured pieces, deserialized as above

- bytes[pos after 0...till the end of stream] -> UCI movements history. Each movement has a length of 2 to 3 bytes and is serialized as follows:
    0 -> Start position
    1 -> End position
    2 -> (Optional) if end position > 128(or last bit flag is set as 1). Unset the first bit on endPosition to obtain the real one(or decrease by 128). Values:
      1 -> Q (Queen promotion)
      2 -> N (Knight promotion)
      3 -> B (Bishop promotion)
      4 -> R (Rook Promotion)
```



### How the board should be deserialized (bytes 4-67)

```
Each table position is represented by a byte.
Of this byte, last 3 bits are used encode the piece type (1-6) or the whole byte is 0 and it means that is an empty position.
After the first 3 bits we have 2 flags
  - &8 -> This bit encodes the owner of the piece, 0 - WHITE_PLAYER, 1 - BLACK PLAYER
  - &16 -> This bit encodes if the piece has been moved, 0 - Not moved, 1 - Moved
```

##### 0 - 6 are pieces types
```
0. NO PIECE
1. PAWN
2. BISHOP
3. KNIGHT
4. ROOK
5. QUEEN
6. KING
```
##### PLAYER TYPES ARE THE FOLLOWING
```
0. WHITE PLAYER
1. BLACK PLAYER
2. UNKNOWN PLAYER (or no player)
```

### Other serialized data inside the structure (bytes 67-)

Past 67 bytes you'll find the pieces removed from the board, the format is the same, you read byte by byte until you find a 0 byte which marks the end of the list of removed table pieces.

After that 0 byte what follows is the history of the match, this has a dynamic length, and each 2 to 3 bytes represent a start and end position on the table between 0-63 and optionally a tag. 

```
0 -> a1
1 -> a2
2 -> a3
...
63 -> f8
```

If the second byte, the end position first bit is set to 1 then it means the movement has a tag, for now only promotion are marked.

If so you must unset the first bit on the end position to get the real end position, and after that read a third byte which will define the tag. 
Tags are described above (1 - "Q", 2 - "N", 3 - "B", 4 - "R")

So each 2 to 3 bytes represent a full movement in UCI form like a2a4, which would be the bytes 8 and 24. Or a promotion like a2a1Q like which would be the bytes 8, 0 and 1. 


### Front-end

Check it here: [ChezzFront](https://github.com/sgatu/chezz-front)

### Run it yourself

Check it here: [Chezz](https://github.com/sgatu/chezz)
