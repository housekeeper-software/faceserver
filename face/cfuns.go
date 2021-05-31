package face

/*
#include <stdio.h>
#include <inttypes.h>

#include "../xface/cJSON.c"
#include "../xface/xface_data.h"

extern void callbackOnCgo(int64_t seq, HobotXFaceImageFeatures* result);

// The gateway function
void go_callback_proxy(int64_t seq, HobotXFaceImageFeatures* result){
	callbackOnCgo(seq,result);
}
*/
import "C"
