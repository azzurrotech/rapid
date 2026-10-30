module rapid.ai

go 1.22.0

require (
	azzurrotech/atp v0.0.0
	azzurrotech/pod v0.0.0
	azzurrotech/shepherd v0.0.0
	azzurrotech/song v0.0.0
)

replace azzurrotech/atp => ../stenella/atp

replace azzurrotech/pod => ../stenella/atp/pod

replace azzurrotech/shepherd => ../stenella/atp/shepherd

replace azzurrotech/song => ../stenella/atp/song
