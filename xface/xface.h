/**
 * Copyright (c) 2018 Horizon Robotics. All rights reserved.
 * @file      xface.h
 * @brief provides the fuction to extract face features from one image
 * @author    chuanyi.yang
 * @email     chuanyi.yang@hobot.cc
 * @version   1.0.0.0
 * @date      2018.06.15
 */
#ifndef XFACE_INTERFACE_HPP
#define XFACE_INTERFACE_HPP

#include <inttypes.h>
#include "xface_data.h"
#include "xface_util.h"

typedef void *HobotXFaceHandle;

/**
 * @brief Get version information
 * @param sdk_version : pointer to the  sdk_version char array
 * @param length : the length of sdk_version
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceGetVersion(char *sdk_version, int length);

/**
 * @brief Get license information
 * @return
 */
HOBOT_XFACE_API const char *HobotXFaceGetLicenseInfo();

/**
 * @brief Get module version information
 * @param model_name : pointer to the  model_version char array
 * @param length : the length of model_version
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceGetModuleVersion(
    const HobotXFaceHandle handle, char *model_version, int length);

/**
 * @brief get error detail by error code
 * @param error_code, [IN]
 * @return
 */
HOBOT_XFACE_API const char *HobotXFaceGetErrorDetail(HobotXFaceErrorCode error_code);

/**
 * @brief Get module version information
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceCreate(HobotXFaceHandle *handle);

/**
 * @brief set config, key value, must be used before HobotXFaceInit
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceSetConfig(const HobotXFaceHandle handle,
                                                     const char *key,
                                                     const char *value);

/**
 * @brief init xface
 * @param config_file, [IN]
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceInit(const HobotXFaceHandle handle);

/**
 * @brief uninit xface
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceFree(HobotXFaceHandle handle);

/**
 * @brief the synchronous API to extract face features from one image
 * @param image, [IN], image info (cannot use pointer because the grammar of
 * golang )
 * @param features, [OUT], have to release the memory manually.
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceExtractFeature(const HobotXFaceHandle handle,
                                                          const HobotXFaceImage image,
                                                          HobotXFaceImageFeatures **features);

/**
 * @brief the synchronous API to extract face features from multi image
 * @param image, [IN], image info (cannot use pointer because the grammar of
 * golang )
 * @param features, [OUT], have to release the memory manually.
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceExtractFeatureMulti(const HobotXFaceHandle handle,
                                                               const HobotXFaceImage *images,
                                                               int imgs_len,
                                                               HobotXFaceImageFeatures ***features);

/**
 * @brief callback function, have to release memory(features) manually
 */
typedef void
(*HobotXFaceCallback)(int64_t seq, HobotXFaceImageFeatures *features);

/**
 * @brief set callback function for the asynchronous API
 * @param cb, [IN]  callback function
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceSetCallback(const HobotXFaceHandle handle,
                                                       const HobotXFaceCallback cb);

/**
 * @brief The asynchronous API to extract face features from one image
 * @param seq, [IN]
 * @param image, [IN], image info (cannot use pointer because the grammar of
 * golang )
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceExtractFeatureAsyn(const HobotXFaceHandle handle,
                                                              int64_t seq,
                                                              const HobotXFaceImage image);

/**
 * @brief release HobotXFaceImageFeatures memory
 * @param features, [IN]
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceRelease(HobotXFaceImageFeatures **features);

/**
 * @brief release HobotXFaceImageFeatures array memory
 * @param features, [IN]
 * @return
 */
HOBOT_XFACE_API HobotXFaceStatus HobotXFaceReleaseMulti(HobotXFaceImageFeatures **features,
                                                        int len);

#endif
