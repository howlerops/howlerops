//go:build duckdb

package duckdb

/*
#include <assert.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#ifndef ARROW_C_DATA_INTERFACE
#define ARROW_C_DATA_INTERFACE

struct ArrowSchema {
  const char* format;
  const char* name;
  const char* metadata;
  int64_t flags;
  int64_t n_children;
  struct ArrowSchema** children;
  struct ArrowSchema* dictionary;
  void (*release)(struct ArrowSchema*);
  void* private_data;
};

struct ArrowArray {
  int64_t length;
  int64_t null_count;
  int64_t offset;
  int64_t n_buffers;
  int64_t n_children;
  const void** buffers;
  struct ArrowArray** children;
  struct ArrowArray* dictionary;
  void (*release)(struct ArrowArray*);
  void* private_data;
};

struct ArrowArrayStream {
  int (*get_schema)(struct ArrowArrayStream*, struct ArrowSchema* out);
  int (*get_next)(struct ArrowArrayStream*, struct ArrowArray* out);
  const char* (*get_last_error)(struct ArrowArrayStream*);
  void (*release)(struct ArrowArrayStream*);
  void* private_data;
};

#endif  // ARROW_C_DATA_INTERFACE

int ArrowSchemaIsReleased(const struct ArrowSchema* schema) {
  return schema == NULL || schema->release == NULL;
}

void ArrowSchemaMarkReleased(struct ArrowSchema* schema) {
  if (schema != NULL) {
    schema->release = NULL;
  }
}

void ArrowSchemaMove(struct ArrowSchema* src, struct ArrowSchema* dest) {
  if (src == NULL || dest == NULL || src == dest) {
    return;
  }
  assert(!ArrowSchemaIsReleased(src));
  memcpy(dest, src, sizeof(struct ArrowSchema));
  ArrowSchemaMarkReleased(src);
}

void ArrowSchemaRelease(struct ArrowSchema* schema) {
  if (schema == NULL || schema->release == NULL) {
    return;
  }
  void (*release_fn)(struct ArrowSchema*) = schema->release;
  release_fn(schema);
  if (schema->release != NULL) {
    ArrowSchemaMarkReleased(schema);
  }
}

int ArrowArrayIsReleased(const struct ArrowArray* array) {
  return array == NULL || array->release == NULL;
}

void ArrowArrayMarkReleased(struct ArrowArray* array) {
  if (array != NULL) {
    array->release = NULL;
  }
}

void ArrowArrayMove(struct ArrowArray* src, struct ArrowArray* dest) {
  if (src == NULL || dest == NULL || src == dest) {
    return;
  }
  assert(!ArrowArrayIsReleased(src));
  memcpy(dest, src, sizeof(struct ArrowArray));
  ArrowArrayMarkReleased(src);
}

void ArrowArrayRelease(struct ArrowArray* array) {
  if (array == NULL || array->release == NULL) {
    return;
  }
  void (*release_fn)(struct ArrowArray*) = array->release;
  release_fn(array);
  if (array->release != NULL) {
    ArrowArrayMarkReleased(array);
  }
}

int ArrowArrayStreamIsReleased(const struct ArrowArrayStream* stream) {
  return stream == NULL || stream->release == NULL;
}

void ArrowArrayStreamMarkReleased(struct ArrowArrayStream* stream) {
  if (stream != NULL) {
    stream->release = NULL;
  }
}

void ArrowArrayStreamMove(struct ArrowArrayStream* src, struct ArrowArrayStream* dest) {
  if (src == NULL || dest == NULL || src == dest) {
    return;
  }
  assert(!ArrowArrayStreamIsReleased(src));
  memcpy(dest, src, sizeof(struct ArrowArrayStream));
  ArrowArrayStreamMarkReleased(src);
}

const char* ArrowArrayStreamGetLastError(struct ArrowArrayStream* stream) {
  if (stream == NULL || stream->get_last_error == NULL) {
    return NULL;
  }
  return stream->get_last_error(stream);
}

void ArrowArrayStreamRelease(struct ArrowArrayStream* stream) {
  if (ArrowArrayStreamIsReleased(stream)) {
    return;
  }
  void (*release_fn)(struct ArrowArrayStream*) = stream->release;
  release_fn(stream);
  if (!ArrowArrayStreamIsReleased(stream)) {
    ArrowArrayStreamMarkReleased(stream);
  }
}
*/
import "C"
