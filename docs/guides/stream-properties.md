# Stream Properties

traverse supports reading and writing OData stream properties - binary content attached to an entity, such as profile photos, document files, or thumbnails.

## Reading a stream property

```go
import (
    "io"
    "os"
    "github.com/jhonsferg/traverse"
)

client := traverse.New(traverse.Config{BaseURL: "https://api.example.com/odata"})

// GET /Products(1)/Photo
reader, contentType, err := client.Entity("Products", 1).
    StreamProperty("Photo").
    Download(ctx)
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

log.Printf("content type: %s", contentType)
io.Copy(os.Stdout, reader)
```

## Writing a stream property

```go
f, err := os.Open("photo.jpg")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

// PUT /Products(1)/Photo
err = client.Entity("Products", 1).
    StreamProperty("Photo").
    Upload(ctx, f, "image/jpeg")
if err != nil {
    log.Fatal(err)
}
```

## Deleting a stream property

```go
// DELETE /Products(1)/Photo
err := client.Entity("Products", 1).
    StreamProperty("Photo").
    Delete(ctx)
```

## Multipart upload

For large files, use chunked multipart upload:

```go
err := client.Entity("Documents", docID).
    StreamProperty("Content").
    UploadMultipart(ctx, traverse.MultipartOptions{
        Reader:      file,
        ContentType: "application/pdf",
        ChunkSize:   4 * 1024 * 1024, // 4 MB chunks
        OnProgress: func(uploaded, total int64) {
            log.Printf("uploaded %d/%d bytes", uploaded, total)
        },
    })
```

## Stream URL

Get the URL for a stream property without downloading:

```go
url, err := client.Entity("Products", 1).
    StreamProperty("Photo").
    URL(ctx)
// url = "https://api.example.com/odata/Products(1)/Photo"
```

## See also

- [CRUD Operations](crud.md)
- [Async Operations](async-operations.md)
