# 关于
-此 Go SDK 基于网宿云存储(WCS)官方API构建

# 版本
`Current version: v1.0.2`

# 运行环境
`Go 1.7及以上。`

# 安装方法
## GitHub安装
`go get -u github.com/Wangsu-Cloud-Storage/wcs-go-sdk-v2/wos`

## 初始化wos客户端

wos客户端(WosClient)是访问WOS服务的Go客户端，它为调用者提供一系列与WOS服务进行交互的接口，用于管理、操作桶（Bucket）和对象（Object）等WOS服务上的资源。
使用WOS Go SDK向WOS发起请求，您需要初始化一个WosClient实例，并根据需要调整客户端配置参数。

**创建客户端**
```
var ak = "*** Provide your Access Key ***"
var sk = "*** Provide your Secret Key ***"
var endpoint = "https://your-endpoint"
 
wosClient, err := wos.New(ak, sk, endpoint, wos.WithRegion("your-region"))
if err == nil {
    // 使用wosClient访问wos
    // ...
 
    // 关闭wosClient
    wosClient.Close()
} else {
    // 解析接口响应的错误code和message
    if wosError, ok := err.(wos.WosError); ok {
        fmt.Println(wosError.StatusCode)
        fmt.Println(wosError.Code)
        fmt.Println(wosError.Message)
    } else {
        fmt.Println(err)
    }
}
```

**配置客户端**

创建客户端时可以通过参数指定采用不同的客户端配置
例如:设置获取响应头的超时时间为30s
```
wosClient, err := wos.New(ak, sk, wos.WithRegion("your-region"), endpoint, wos.WithHeaderTimeout(30))
```

**更多参数见下表**
| 配置方式 | 描述 | 建议值 |
| -- | -- | -- |
| WithSignature(signature SignatureType)	| 签名鉴权形式。默认为wos鉴权(wos.SignatureWos),也可配置为aws-v2(wos.SignatureV2)、aws-v4(wos.SignatureV4)	| N/A
| WithSslVerifyAndPemCerts(sslVerify bool, pemCerts []byte)	| 配置验证服务端证书的参数。默认为不验证。	| N/A
| WithHeaderTimeout(headerTimeout int)	| 配置获取响应头的超时时间。默认为60秒。	| 10，60
| WithMaxConnections(maxIdleConns int)	| 配置允许最大HTTP空闲连接数。默认为1000。	| N/A
| WithConnectTimeout(connectTimeout int)	| 配置建立HTTP/HTTPS连接的超时时间（单位：秒）。默认为60秒。	| 10，60
| WithSocketTimeout(socketTimeout int)	| 配置读写数据的超时时间（单位：秒）。默认为60秒。	| 10，60
| WithIdleConnTimeout(idleConnTimeout int)	| 配置空闲的HTTP连接在连接池中的超时时间（单位：秒）。默认为30秒。	| 默认
| WithMaxRetryCount(maxRetryCount int)	| 配置HTTP/HTTPS连接异常时的请求重试次数。默认为3次。	| 1，5
| WithProxyUrl(proxyUrl string)	| 配置HTTP代理。	| N/A
| WithHttpTransport(transport *http.Transport)	| 配置自定义的Transport。	| 默认
| WithRequestContext(ctx context.Context)	| 配置每次HTTP请求的上下文。	| N/A
| WithMaxRedirectCount(maxRedirectCount int)	| 配置HTTP/HTTPS请求重定向的最大次数。默认为3次。	| 1，5
| WithRegion(region string) | 配置S3所在region | default-region
| WithPathStyle(pathStyle boolean)| 是否使用路径模式,关闭时使用使用bucketName.endpoint格式URL访问服务；开启时使用endpoint/bucketName格式URL访问服务。默认关闭|默认

# 快速使用
## 获取存储空间列表（List Bucket）
```
var ak = "*** Provide your Access Key ***"
var sk = "*** Provide your Secret Key ***"
var endpoint = "https://your-endpoint"
 
wosClient, _ := wos.New(ak, sk, endpoint, wos.WithRegion("your-region"))
 
input := &wos.ListBucketsInput{}
input.QueryLocation = true
output, err := wosClient.ListBuckets(input)
if err == nil {
    fmt.Printf("StatusCode:%d, RequestId:%s\n", output.StatusCode, output.RequestId)
    fmt.Printf("Owner.DisplayName:%s, Owner.ID:%s\n", output.Owner.DisplayName, output.Owner.ID)
    for index, val := range output.Buckets {
        fmt.Printf("Bucket[%d]-Name:%s,CreationDate:%s,EndPoint:%s,Region:%s\n", index, val.Name, val.CreationDate, val.Endpoint, val.Region)
    }
} else {
    // 解析接口响应的错误code和message
    if wosError, ok := err.(wos.WosError); ok {
        fmt.Println(wosError.StatusCode)
        fmt.Println(wosError.Code)
        fmt.Println(wosError.Message)
    } else {
        fmt.Println(err)
    }
}
```

## 获取空间文件列表（List Objects）
```
var ak = "*** Provide your Access Key ***"
var sk = "*** Provide your Secret Key ***"
var endpoint = "https://your-endpoint"
 
wosClient, _ := wos.New(ak, sk, endpoint, wos.WithRegion("your-region"))
 
input := &wos.ListObjectsInput{}
input.Bucket = bucketName
//  input.Prefix = "src/"
output, err := wosClient.ListObjects(input)
if err == nil {
    fmt.Printf("StatusCode:%d, RequestId:%s,OwnerId:%s,OwnerName:%s,\n", output.StatusCode, output.RequestId, output.Owner.ID, output.Owner.DisplayName)
    for index, val := range output.Contents {
        fmt.Printf("Content[%d]-ETag:%s, Key:%s, LastModified:%s, Size:%d, StorageClass:%s\n",
            index, val.ETag, val.Key, val.LastModified, val.Size, val.StorageClass)
    }
} else {
    // 解析接口响应的错误code和message
    if wosError, ok := err.(wos.WosError); ok {
        fmt.Println(wosError.StatusCode)
        fmt.Println(wosError.Code)
        fmt.Println(wosError.Message)
    } else {
        fmt.Println(err)
    }
}
```

## 上传文件（Put Object）
```
var ak = "*** Provide your Access Key ***"
var sk = "*** Provide your Secret Key ***"
var endpoint = "https://your-endpoint"
 
wosClient, _ := wos.New(ak, sk, endpoint, wos.WithRegion("your-region"))
 
input := &wos.PutObjectInput{}
input.Bucket = bucketName
input.Key = objectKey
input.Metadata = map[string]string{"meta": "value"}
input.Body = strings.NewReader("Hello WOS")
output, err := wosClient.PutObject(input)
if err == nil {
    fmt.Printf("StatusCode:%d, RequestId:%s\n", output.StatusCode, output.RequestId)
    fmt.Printf("ETag:%s", output.ETag)
} else {
    // 解析接口响应的错误code和message
    if wosError, ok := err.(wos.WosError); ok {
        fmt.Println(wosError.StatusCode)
        fmt.Println(wosError.Code)
        fmt.Println(wosError.Message)
    } else {
        fmt.Println(err)
    }
}
```

## 下载文件 (Get Object)
```
var ak = "*** Provide your Access Key ***"
var sk = "*** Provide your Secret Key ***"
var endpoint = "https://your-endpoint"
 
wosClient, _ := wos.New(ak, sk, endpoint, wos.WithRegion("your-region"))
 
input := &wos.GetObjectInput{}
input.Bucket = bucketName
input.Key = objectKey
output, err := wosClient.GetObject(input)
if err == nil {
    defer output.Body.Close()
    fmt.Printf("StatusCode:%d, RequestId:%s\n", output.StatusCode, output.RequestId)
    fmt.Printf("StorageClass:%s, ETag:%s, ContentType:%s, ContentLength:%d, LastModified:%s\n",
        output.StorageClass, output.ETag, output.ContentType, output.ContentLength, output.LastModified)
    p := make([]byte, 1024)
    var readErr error
    var readCount int
    for {
        readCount, readErr = output.Body.Read(p)
        if readCount > 0 {
            fmt.Printf("%s", p[:readCount])
        }
        if readErr != nil {
            break
        }
    }
} else {
    // 解析接口响应的错误code和message
    if wosError, ok := err.(wos.WosError); ok {
        fmt.Println(wosError.StatusCode)
        fmt.Println(wosError.Code)
        fmt.Println(wosError.Message)
    } else {
        fmt.Println(err)
    }
}
```

## 删除文件(Delete Object)
```
var ak = "*** Provide your Access Key ***"
var sk = "*** Provide your Secret Key ***"
var endpoint = "https://your-endpoint"
 
wosClient, _ := wos.New(ak, sk, endpoint, wos.WithRegion("your-region"))
 
input := &wos.DeleteObjectInput{}
input.Bucket = bucketName
input.Key = objectKey
output, err := wosClient.DeleteObject(input)
if err == nil {
    fmt.Printf("StatusCode:%d, RequestId:%s\n", output.StatusCode, output.RequestId)
} else {
    // 解析接口响应的错误code和message
    if wosError, ok := err.(wos.WosError); ok {
        fmt.Println(wosError.StatusCode)
        fmt.Println(wosError.Code)
        fmt.Println(wosError.Message)
    } else {
        fmt.Println(err)
    }
}
```

## 分段上传
### 简单分段上传
参考 `example/simple_multipart_upload_sample.go`

### 块并发分段上传
参考 `example/multipart_upload_sample.go`

# 功能列举
## 基础功能 
示例位于`wos_go_sample.go`

| 功能 | 名称 |
| -- | -- |
| 列举空间	| ListBuckets
| 判断空间是否存在	| HeadBucket
| 设置空间的生命周期	| SetBucketLifecycleConfiguration
| 获取空间的生命周期	| GetBucketLifecycleConfiguration
| 删除空间的生命周期	| deleteBucketLifecycleConfiguration
| 列举文件	| ListObjects
| 列举文件v2	| ListObjectV2
| 列举分片文件	| ListMultipartUploads
| 删除对象	| DeleteObject
| 批量删除对象	| DeleteObjects
| 取回归档存储对象	| RestoreObject
| 初始化分段上传任务	| InitiateMultipartUpload
| 上传分段	| UploadPart
| 复制分段	| CopyPart
| 列举已上传分段	| ListParts
| 取消分段上传任务	| AbortMultipartUpload
| 上传对象	| PutObject
| 上传文件	| PutFile
| 判断对象是否存在	| HeadObject
| 获取对象元数据	| GetObjectMetadata
| 下载对象	| GetObject
| 获取对象avinfo	| GetAvinfo

## 更多示例
| 示例文件 | 示例内容 |
| -- | -- |
| 分段并发复制大对象	| concurrent_copy_part_sample.go
| 分段并发下载大对象	| concurrent_download_object_sample.go
| 分段并发上传大对象	| multipart_upload_sample.go
| 创建文件夹	| create_folder_sample.go
| 批量删除对象	| delete_objects_sample.go
| 下载文件	| download_sample.go
| 列举空间中对象	| list_objects_sample.go
| 列举文件夹中对象	| list_objects_in_folder_sample.go
| 对象元数据操作	| object_meta_sample.go
| 对象简单操作示例	| object_operations_sample.go
| 简易分片上传	| simple_multipart_upload_sample.go
| 临时鉴权操作示例	| temporary_signature_sample.go
