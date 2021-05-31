#ifndef XFACE_UTIL_H
#define XFACE_UTIL_H

#ifdef _WIN32
#ifdef XFACE_WIN_VER // build
#define BUILD_XFACE_EXPORT __declspec(dllexport)
#else  // interface
#define BUILD_XFACE_EXPORT __declspec(dllimport)
#endif // XFACE_WIN_VER
#elif defined(__GNUC__) && __GNUC__ >= 4
#define BUILD_XFACE_EXPORT __attribute__((visibility("default")))
#else
#define BUILD_XFACE_EXPORT
#endif

#ifdef __cplusplus
#define HOBOT_XFACE_API  extern "C" BUILD_XFACE_EXPORT
#else
#define HOBOT_XFACE_API BUILD_XFACE_EXPORT
#endif

HOBOT_XFACE_API void XFaceConvertToRGB_Gray8(unsigned char *buffer_gray,
                                             unsigned char *buffer_rgb,
                                             int width,
                                             int height,
                                             int ratio);
HOBOT_XFACE_API void XFaceConvertToRGB_IYUV420(unsigned char *buffer_yuv420p,
                                               unsigned char *buffer_rgb,
                                               int width,
                                               int height,
                                               int ratio);
HOBOT_XFACE_API void XFaceConvertToRGB_NV21(unsigned char *buffer_nv21,
                                            unsigned char *buffer_rgb,
                                            int width,
                                            int height,
                                            int ratio);
HOBOT_XFACE_API void XFaceConvertToRGB_NV12(unsigned char *buffer_nv12,
                                            unsigned char *buffer_rgb,
                                            int width,
                                            int height,
                                            int ratio);
HOBOT_XFACE_API void XFaceConvertToRGB_YUY2(unsigned char *buffer_yuy2,
                                            unsigned char *buffer_rgb,
                                            int width,
                                            int height,
                                            int ratio);

#endif  //
