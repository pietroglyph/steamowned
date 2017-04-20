# steamowned
Check what games a subset of an arbitrary amount of steam users share... concurrently... with a bloom filter...

## Usage
1. Install Go
2. Setup your environment
3. Run this:
```bash
$ git clone https://github.com/pietroglyph/steamowned
$ cd steamowned
$ go install
$ $GOPATH/bin/steamowned -api-key [YOUR STEAM API KEY]
```
4. Point your browser to http://127.0.0.1?players=[STEAMID64]|[ANOTHER STEAMID64]
## TODO
- [ ] Human readable names for games
- [ ] A slick web interface of some sort
