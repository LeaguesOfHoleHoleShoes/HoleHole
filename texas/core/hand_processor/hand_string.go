// Code generated by "stringer -type=HandType"; DO NOT EDIT.

package hand_processor

import "strconv"

const _HandType_name = "HandOfDZHandOfYDHandOfLDHandOfST3HandOfSZHandOfTHHandOfHLHandOfST4HandOfTHSHandOfHJTHS"

var _HandType_index = [...]uint8{0, 8, 16, 24, 33, 41, 49, 57, 66, 75, 86}

func (i HandType) String() string {
	if i < 0 || i >= HandType(len(_HandType_index)-1) {
		return "HandType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _HandType_name[_HandType_index[i]:_HandType_index[i+1]]
}
