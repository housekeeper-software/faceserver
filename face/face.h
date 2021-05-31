#ifndef FACE_H_
#define FACE_H_

#include <inttypes.h>
#include "../xface/xface_data.h"

typedef void (*Callback)(int64_t seq, HobotXFaceImageFeatures *result);

int InitFaceLib(const char *conf, const char *model_conf, Callback callback);

int DoFeature(int64_t seq, int predict_mode, int max_face_count, const void *data, int length);

void UnInitFaceLib();

#endif //FACE_H_

