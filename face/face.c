#include "face.h"
#include <stdio.h>
#include <malloc.h>
#include <stdlib.h>
#include <math.h>
#include "../xface/xface.h"
#include "../xface/cJSON.h"

/*
{
      "face_quality":"2",
      "alphadet_instance_count":"8",
      "min_area":"100",
      "min_face_width":"80",
      "min_face_height":"80",
      "face_keypoint":"1",
      "face_metric_feature":"1",
      "feature_instance_count":"4",
      "pose_pitch_min":"-90.0",
      "pose_pitch_max":"-90.0",
      "pose_yaw_min":"-90.0",
      "pose_yaw_max":"90.0",
      "pose_roll_min":"-180.0",
      "pose_roll_max":"180.0",
      "face_liveness":"1"
   }
 */

enum ContentType {
  Type_File = 0,
  Type_Base64
};

enum PrivateError {
  PError_Parameters = -1,
  PError_File_Not_Found = -2,
  PError_NO_Feature = -3
};

struct Face {
  HobotXFaceHandle xface_handle;
  Callback callback;
};

static struct Face *gInstance = NULL;

static void OnFeature(int64_t seq, HobotXFaceImageFeatures *features) {
  if (gInstance == NULL)
    return;
  gInstance->callback(seq, features);
  HobotXFaceRelease(&features);
}

int InitXFace(const char *conf, const char *model_conf, HobotXFaceHandle *handle) {
  HobotXFaceStatus result;
  HobotXFaceHandle xface_handle = NULL;
  result = HobotXFaceCreate(&xface_handle);
  if (result != ErrorCode_OK) {
    fprintf(stderr,
            "HobotXFaceCreate failed:%d, desc:%s\n",
            result,
            HobotXFaceGetErrorDetail((HobotXFaceErrorCode) result));
    return result;
  }

  result = HobotXFaceSetConfig(xface_handle, "model_conf", model_conf);
  if (result != ErrorCode_OK) {
    HobotXFaceFree(xface_handle);
    fprintf(stderr,
            "HobotXFaceSetConfig failed:%d,desc:%s\n",
            result,
            HobotXFaceGetErrorDetail((HobotXFaceErrorCode) result));
    return result;
  }

  cJSON *root = cJSON_Parse(conf);
  if (root) {
    cJSON *c = root->child;
    while (c) {
      result = HobotXFaceSetConfig(xface_handle, c->string, c->valuestring);
      if (result != ErrorCode_OK) {
        fprintf(stderr, "HobotXFaceSetConfig(%s,%s) %d,desc:%s\n",
                c->string,
                c->valuestring,
                result,
                HobotXFaceGetErrorDetail((HobotXFaceErrorCode) result));
      } else {
        fprintf(stdout, "HobotXFaceSetConfig(%s,%s) success\n", c->string, c->valuestring);
      }
      c = c->next;
    }
  }
  cJSON_Delete(root);

  result = HobotXFaceInit(xface_handle);
  if (result != ErrorCode_OK) {
    fprintf(stderr,
            "HobotXFaceInit failed:%d,desc:%s\n",
            result,
            HobotXFaceGetErrorDetail((HobotXFaceErrorCode) result));
    HobotXFaceFree(xface_handle);
    return result;
  }
  char sdk_version[50] = {0};
  HobotXFaceGetVersion(sdk_version, 50);
  fprintf(stdout, "HobotXFaceGetVersion %s\n", sdk_version);
  char model_version[50] = {0};
  HobotXFaceGetModuleVersion(xface_handle, model_version, 50);
  fprintf(stdout, "HobotXFaceGetModuleVersion:%s\n", model_version);
  HobotXFaceSetCallback(xface_handle, &OnFeature);
  *handle = xface_handle;
  return ErrorCode_OK;
}

int InitFaceLib(const char *conf, const char *model_conf, Callback callback) {
  setvbuf(stdout, NULL, _IONBF, 0);
  setvbuf(stderr, NULL, _IONBF, 0);

  if (gInstance) {
    return ErrorCode_OK;
  }
  gInstance = malloc(sizeof(struct Face));
  memset(gInstance, 0, sizeof(struct Face));
  gInstance->callback = callback;
  return InitXFace(conf, model_conf, &gInstance->xface_handle);
}

void UnInitFaceLib() {
  if (gInstance) {
    if (gInstance->xface_handle) {
      HobotXFaceFree(gInstance->xface_handle);
    }
    free(gInstance);
    gInstance = NULL;
  }
}

int DoFeature(int64_t seq, int predict_mode, int max_face_count, const void *data, int length) {
  if (!gInstance) {
    return ErrorCode_Uninit;
  }

  int predict = PredictMode_Metric | PredictMode_Quality;
  if (predict_mode > 0) {
    predict = predict_mode;
  }
  int face_count = 1;
  if (max_face_count > 1) {
    face_count = max_face_count;
  }
  HobotXFaceImage image;
  memset(&image, 0, sizeof(HobotXFaceImage));
  image.buf_ = (unsigned char *) (data);
  image.buf_len_ = length;
  image.buf_type_ = ImgType_None;
  image.predict_mode_ = predict;
  image.max_face_count_ = face_count;
  return HobotXFaceExtractFeatureAsyn(gInstance->xface_handle, seq, image);
}
