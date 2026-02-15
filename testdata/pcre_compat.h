#ifndef PCRE_COMPAT_H
#define PCRE_COMPAT_H

/*
 * PCRE1-to-PCRE2 compatibility shim.
 *
 * The C ccze source uses the PCRE1 API, but modern distros only ship PCRE2.
 * This header maps the PCRE1 calls used by ccze onto their PCRE2 equivalents
 * so the project can be built without the legacy libpcre3-dev package.
 */

#define PCRE2_CODE_UNIT_WIDTH 8
#include <pcre2.h>
#include <stdlib.h>
#include <string.h>

typedef pcre2_code pcre;
typedef struct { int dummy; } pcre_extra;

#define PCRE_CASELESS PCRE2_CASELESS

static inline pcre *pcre_compile(const char *pattern, int options,
                                  const char **errptr, int *erroffset,
                                  const void *tableptr) {
    int errorcode; PCRE2_SIZE eoffset; (void)tableptr;
    pcre2_code *re = pcre2_compile((PCRE2_SPTR)pattern, PCRE2_ZERO_TERMINATED,
                                    (uint32_t)options, &errorcode, &eoffset, NULL);
    if (!re && errptr) {
        static char errbuf[256];
        pcre2_get_error_message(errorcode, (PCRE2_UCHAR *)errbuf, sizeof(errbuf));
        *errptr = errbuf;
    }
    if (erroffset) *erroffset = (int)eoffset;
    return re;
}

static inline pcre_extra *pcre_study(pcre *re, int options, const char **errptr) {
    (void)re; (void)options; (void)errptr; return NULL;
}

static inline int pcre_exec(const pcre *re, const pcre_extra *extra,
                             const char *subject, int length, int startoffset,
                             int options, int *ovector, int ovecsize) {
    (void)extra;
    pcre2_match_data *md = pcre2_match_data_create(ovecsize / 3, NULL);
    if (!md) return -1;
    int rc = pcre2_match(re, (PCRE2_SPTR)subject, (PCRE2_SIZE)length,
                          (PCRE2_SIZE)startoffset, (uint32_t)options, md, NULL);
    if (rc > 0) {
        PCRE2_SIZE *ov = pcre2_get_ovector_pointer(md);
        int pairs = rc * 2;
        if (pairs > ovecsize) pairs = ovecsize;
        for (int i = 0; i < pairs; i++) ovector[i] = (int)ov[i];
    }
    pcre2_match_data_free(md);
    return rc;
}

static inline int pcre_get_substring(const char *subject, int *ovector,
                                      int stringcount, int stringnumber,
                                      const char **stringptr) {
    (void)stringcount;
    int start = ovector[2 * stringnumber];
    int end = ovector[2 * stringnumber + 1];
    if (start < 0) { *stringptr = NULL; return -1; }
    int len = end - start;
    char *buf = malloc(len + 1);
    if (!buf) return -1;
    memcpy(buf, subject + start, len);
    buf[len] = '\0';
    *stringptr = buf;
    return len;
}

static inline void pcre_free_substring(const char *str) { free((void *)str); }
static inline void pcre_free(void *ptr) { pcre2_code_free((pcre2_code *)ptr); }

#endif
