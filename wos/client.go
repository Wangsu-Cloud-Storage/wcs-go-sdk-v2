package wos

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// WosClient defines WOS client.
type WosClient struct {
	conf       *config
	httpClient *http.Client
}

// New creates a new WosClient instance.
func New(ak, sk, endpoint string, configurers ...configurer) (*WosClient, error) {
	conf := &config{endpoint: endpoint}
	conf.securityProviders = make([]securityProvider, 0, 3)
	conf.securityProviders = append(conf.securityProviders, NewBasicSecurityProvider(ak, sk, ""))

	conf.maxRetryCount = -1
	conf.maxRedirectCount = -1
	for _, configurer := range configurers {
		configurer(conf)
	}

	if err := conf.initConfigWithDefault(); err != nil {
		return nil, err
	}
	err := conf.getTransport()
	if err != nil {
		return nil, err
	}

	if isWarnLogEnabled() {
		info := make([]string, 3)
		info[0] = fmt.Sprintf("[WOS SDK Version=%s", wosSdkVersion)
		info[1] = fmt.Sprintf("Endpoint=%s", conf.endpoint)
		accessMode := "Virtual Hosting"
		if conf.pathStyle {
			accessMode = "Path"
		}
		info[2] = fmt.Sprintf("Access Mode=%s]", accessMode)
		doLog(LEVEL_WARN, strings.Join(info, "];["))
	}
	doLog(LEVEL_DEBUG, "Create wosclient with config:\n%s\n", conf)
	wosClient := &WosClient{conf: conf, httpClient: &http.Client{Transport: conf.transport, CheckRedirect: checkRedirectFunc}}
	return wosClient, nil
}

// Refresh refreshes ak, sk and securityToken for wosClient.
func (wosClient WosClient) Refresh(ak, sk, securityToken string) {
	for _, sp := range wosClient.conf.securityProviders {
		if bsp, ok := sp.(*BasicSecurityProvider); ok {
			bsp.refresh(strings.TrimSpace(ak), strings.TrimSpace(sk))
			break
		}
	}
}

func (wosClient WosClient) getSecurity() securityHolder {
	if wosClient.conf.securityProviders != nil {
		for _, sp := range wosClient.conf.securityProviders {
			if sp == nil {
				continue
			}
			sh := sp.getSecurity()
			if sh.ak != "" && sh.sk != "" {
				return sh
			}
		}
	}
	return emptySecurityHolder
}

// Close closes WosClient.
func (wosClient WosClient) Close() {
	wosClient.httpClient = nil
	wosClient.conf.transport.CloseIdleConnections()
	wosClient.conf = nil
}

// ListBuckets lists buckets.
//
// You can use this API to obtain the bucket list. In the list, bucket names are displayed in lexicographical order.
func (wosClient WosClient) ListBuckets(input *ListBucketsInput, extensions ...extensionOptions) (output *ListBucketsOutput, err error) {
	if input == nil {
		input = &ListBucketsInput{}
	}
	output = &ListBucketsOutput{}
	err = wosClient.doActionWithoutBucket("ListBuckets", HTTP_GET, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// ListObjects lists objects in a bucket.
//
// You can use this API to list objects in a bucket. By default, a maximum of 1000 objects are listed.
func (wosClient WosClient) ListObjects(input *ListObjectsInput, extensions ...extensionOptions) (output *ListObjectsOutput, err error) {
	if input == nil {
		return nil, errors.New("ListObjectsInput is nil")
	}
	output = &ListObjectsOutput{}
	err = wosClient.doActionWithBucket("ListObjects", HTTP_GET, input.Bucket, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

func (wosClient WosClient) ListObjectV2(input *ListObjectsV2Input, extensions ...extensionOptions) (output *ListObjectsV2Output, err error) {
	if input == nil {
		return nil, errors.New("ListObjectsInput is nil")
	}
	output = &ListObjectsV2Output{}
	err = wosClient.doActionWithBucket("ListObjects", HTTP_GET, input.Bucket, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// ListMultipartUploads lists the multipart uploads.
//
// You can use this API to list the multipart uploads that are initialized but not combined or aborted in a specified bucket.
func (wosClient WosClient) ListMultipartUploads(input *ListMultipartUploadsInput, extensions ...extensionOptions) (output *ListMultipartUploadsOutput, err error) {
	if input == nil {
		return nil, errors.New("ListMultipartUploadsInput is nil")
	}
	output = &ListMultipartUploadsOutput{}
	err = wosClient.doActionWithBucket("ListMultipartUploads", HTTP_GET, input.Bucket, input, output, extensions)
	if err != nil {
		output = nil
	} else if output.EncodingType == "url" {
		err = decodeListMultipartUploadsOutput(output)
		if err != nil {
			doLog(LEVEL_ERROR, "Failed to get ListMultipartUploadsOutput with error: %v.", err)
			output = nil
		}
	}
	return
}

// HeadBucket checks whether a bucket exists.
//
// You can use this API to check whether a bucket exists.
func (wosClient WosClient) HeadBucket(bucketName string, extensions ...extensionOptions) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = wosClient.doActionWithBucket("HeadBucket", HTTP_HEAD, bucketName, defaultSerializable, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// HeadObject checks whether an object exists.
//
// You can use this API to check whether an object exists.
func (wosClient WosClient) HeadObject(input *HeadObjectInput, extensions ...extensionOptions) (output *BaseModel, err error) {
	if input == nil {
		return nil, errors.New("HeadObjectInput is nil")
	}
	output = &BaseModel{}
	err = wosClient.doActionWithBucketAndKey("HeadObject", HTTP_HEAD, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// SetObjectMetadata sets object metadata.
func (wosClient WosClient) SetObjectMetadata(input *SetObjectMetadataInput, extensions ...extensionOptions) (output *SetObjectMetadataOutput, err error) {
	output = &SetObjectMetadataOutput{}
	err = wosClient.doActionWithBucketAndKey("SetObjectMetadata", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseSetObjectMetadataOutput(output)
	}
	return
}

// SetBucketLifecycleConfiguration sets lifecycle rules for a bucket.
//
// You can use this API to set lifecycle rules for a bucket, to periodically transit
// storage classes of objects and delete objects in the bucket.
func (wosClient WosClient) SetBucketLifecycleConfiguration(input *SetBucketLifecycleConfigurationInput, extensions ...extensionOptions) (output *BaseModel, err error) {
	if input == nil {
		return nil, errors.New("SetBucketLifecycleConfigurationInput is nil")
	}
	output = &BaseModel{}
	err = wosClient.doActionWithBucket("SetBucketLifecycleConfiguration", HTTP_PUT, input.Bucket, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// GetBucketLifecycleConfiguration gets lifecycle rules of a bucket.
//
// You can use this API to obtain the lifecycle rules of a bucket.
func (wosClient WosClient) GetBucketLifecycleConfiguration(bucketName string, extensions ...extensionOptions) (output *GetBucketLifecycleConfigurationOutput, err error) {
	output = &GetBucketLifecycleConfigurationOutput{}
	err = wosClient.doActionWithBucket("GetBucketLifecycleConfiguration", HTTP_GET, bucketName, newSubResourceSerial(SubResourceLifecycle), output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// DeleteBucketLifecycleConfiguration deletes lifecycle rules of a bucket.
//
// You can use this API to delete all lifecycle rules of a bucket.
func (wosClient WosClient) DeleteBucketLifecycleConfiguration(bucketName string, extensions ...extensionOptions) (output *BaseModel, err error) {
	output = &BaseModel{}
	err = wosClient.doActionWithBucket("DeleteBucketLifecycleConfiguration", HTTP_DELETE, bucketName, newSubResourceSerial(SubResourceLifecycle), output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// DeleteObject deletes an object.
//
// You can use this API to delete an object from a specified bucket.
func (wosClient WosClient) DeleteObject(input *DeleteObjectInput, extensions ...extensionOptions) (output *DeleteObjectOutput, err error) {
	if input == nil {
		return nil, errors.New("DeleteObjectInput is nil")
	}
	output = &DeleteObjectOutput{}
	err = wosClient.doActionWithBucketAndKey("DeleteObject", HTTP_DELETE, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// DeleteObjects deletes objects in a batch.
//
// You can use this API to batch delete objects from a specified bucket.
func (wosClient WosClient) DeleteObjects(input *DeleteObjectsInput, extensions ...extensionOptions) (output *DeleteObjectsOutput, err error) {
	if input == nil {
		return nil, errors.New("DeleteObjectsInput is nil")
	}
	output = &DeleteObjectsOutput{}
	err = wosClient.doActionWithBucket("DeleteObjects", HTTP_POST, input.Bucket, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// RestoreObject restores an object.
func (wosClient WosClient) RestoreObject(input *RestoreObjectInput, extensions ...extensionOptions) (output *BaseModel, err error) {
	if input == nil {
		return nil, errors.New("RestoreObjectInput is nil")
	}
	output = &BaseModel{}
	err = wosClient.doActionWithBucketAndKey("RestoreObject", HTTP_POST, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// GetObjectMetadata gets object metadata.
//
// You can use this API to send a HEAD request to the object of a specified bucket to obtain its metadata.
func (wosClient WosClient) GetObjectMetadata(input *GetObjectMetadataInput, extensions ...extensionOptions) (output *GetObjectMetadataOutput, err error) {
	if input == nil {
		return nil, errors.New("GetObjectMetadataInput is nil")
	}
	output = &GetObjectMetadataOutput{}
	err = wosClient.doActionWithBucketAndKey("GetObjectMetadata", HTTP_HEAD, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseGetObjectMetadataOutput(output)
	}
	return
}

func (wosClient WosClient) GetAvinfo(input *GetAvinfoInput, extensions ...extensionOptions) (output *GetAvinfoOutput, err error) {
	if input == nil {
		return nil, errors.New("GetAvinfoInput is nil")
	}
	output = &GetAvinfoOutput{}
	err = wosClient.doActionWithBucketAndKey("GetObject", HTTP_GET, input.Bucket, input.Key, newSubResourceSerial(SubResourceAvinfo), output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// GetObject downloads object.
//
// You can use this API to download an object in a specified bucket.
func (wosClient WosClient) GetObject(input *GetObjectInput, extensions ...extensionOptions) (output *GetObjectOutput, err error) {
	if input == nil {
		return nil, errors.New("GetObjectInput is nil")
	}
	output = &GetObjectOutput{}
	err = wosClient.doActionWithBucketAndKey("GetObject", HTTP_GET, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseGetObjectOutput(output)
	}
	return
}

// PutObject uploads an object to the specified bucket.
func (wosClient WosClient) PutObject(input *PutObjectInput, extensions ...extensionOptions) (output *PutObjectOutput, err error) {
	if input == nil {
		return nil, errors.New("PutObjectInput is nil")
	}

	if input.ContentType == "" && input.Key != "" {
		if contentType, ok := mimeTypes[strings.ToLower(input.Key[strings.LastIndex(input.Key, ".")+1:])]; ok {
			input.ContentType = contentType
		}
	}
	output = &PutObjectOutput{}
	var repeatable bool
	if input.Body != nil {
		_, repeatable = input.Body.(*strings.Reader)
		if input.ContentLength > 0 {
			input.Body = &readerWrapper{reader: input.Body, totalCount: input.ContentLength}
		}
	}
	if repeatable {
		err = wosClient.doActionWithBucketAndKey("PutObject", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	} else {
		err = wosClient.doActionWithBucketAndKeyUnRepeatable("PutObject", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	}
	if err != nil {
		output = nil
	} else {
		ParsePutObjectOutput(output)
	}
	return
}

func (wosClient WosClient) getContentType(input *PutObjectInput, sourceFile string) (contentType string) {
	if contentType, ok := mimeTypes[strings.ToLower(input.Key[strings.LastIndex(input.Key, ".")+1:])]; ok {
		return contentType
	}
	if contentType, ok := mimeTypes[strings.ToLower(sourceFile[strings.LastIndex(sourceFile, ".")+1:])]; ok {
		return contentType
	}
	return
}

func (wosClient WosClient) isGetContentType(input *PutObjectInput) bool {
	if input.ContentType == "" && input.Key != "" {
		return true
	}
	return false
}

// PutFile uploads a file to the specified bucket.
func (wosClient WosClient) PutFile(input *PutFileInput, extensions ...extensionOptions) (output *PutObjectOutput, err error) {
	if input == nil {
		return nil, errors.New("PutFileInput is nil")
	}

	var body io.Reader
	sourceFile := strings.TrimSpace(input.SourceFile)
	if sourceFile != "" {
		fd, _err := os.Open(sourceFile)
		if _err != nil {
			err = _err
			return nil, err
		}
		defer func() {
			errMsg := fd.Close()
			if errMsg != nil {
				doLog(LEVEL_WARN, "Failed to close file with reason: %v", errMsg)
			}
		}()

		stat, _err := fd.Stat()
		if _err != nil {
			err = _err
			return nil, err
		}
		fileReaderWrapper := &fileReaderWrapper{filePath: sourceFile}
		fileReaderWrapper.reader = fd
		if input.ContentLength > 0 {
			if input.ContentLength > stat.Size() {
				input.ContentLength = stat.Size()
			}
			fileReaderWrapper.totalCount = input.ContentLength
		} else {
			fileReaderWrapper.totalCount = stat.Size()
		}
		body = fileReaderWrapper
	}

	_input := &PutObjectInput{}
	_input.PutObjectBasicInput = input.PutObjectBasicInput
	_input.Body = body

	if wosClient.isGetContentType(_input) {
		_input.ContentType = wosClient.getContentType(_input, sourceFile)
	}

	output = &PutObjectOutput{}
	err = wosClient.doActionWithBucketAndKey("PutFile", HTTP_PUT, _input.Bucket, _input.Key, _input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParsePutObjectOutput(output)
	}
	return
}

// CopyObject creates a copy for an existing object.
//
// You can use this API to create a copy for an object in a specified bucket.
func (wosClient WosClient) CopyObject(input *CopyObjectInput, extensions ...extensionOptions) (output *CopyObjectOutput, err error) {
	if input == nil {
		return nil, errors.New("CopyObjectInput is nil")
	}

	if strings.TrimSpace(input.CopySourceBucket) == "" {
		return nil, errors.New("Source bucket is empty")
	}
	if strings.TrimSpace(input.CopySourceKey) == "" {
		return nil, errors.New("Source key is empty")
	}

	output = &CopyObjectOutput{}
	err = wosClient.doActionWithBucketAndKey("CopyObject", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseCopyObjectOutput(output)
	}
	return
}

// AbortMultipartUpload aborts a multipart upload in a specified bucket by using the multipart upload ID.
func (wosClient WosClient) AbortMultipartUpload(input *AbortMultipartUploadInput, extensions ...extensionOptions) (output *BaseModel, err error) {
	if input == nil {
		return nil, errors.New("AbortMultipartUploadInput is nil")
	}
	if input.UploadId == "" {
		return nil, errors.New("UploadId is empty")
	}
	output = &BaseModel{}
	err = wosClient.doActionWithBucketAndKey("AbortMultipartUpload", HTTP_DELETE, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// InitiateMultipartUpload initializes a multipart upload.
func (wosClient WosClient) InitiateMultipartUpload(input *InitiateMultipartUploadInput, extensions ...extensionOptions) (output *InitiateMultipartUploadOutput, err error) {
	if input == nil {
		return nil, errors.New("InitiateMultipartUploadInput is nil")
	}

	if input.ContentType == "" && input.Key != "" {
		if contentType, ok := mimeTypes[strings.ToLower(input.Key[strings.LastIndex(input.Key, ".")+1:])]; ok {
			input.ContentType = contentType
		}
	}

	output = &InitiateMultipartUploadOutput{}
	err = wosClient.doActionWithBucketAndKey("InitiateMultipartUpload", HTTP_POST, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseInitiateMultipartUploadOutput(output)
		if output.EncodingType == "url" {
			err = decodeInitiateMultipartUploadOutput(output)
			if err != nil {
				doLog(LEVEL_ERROR, "Failed to get InitiateMultipartUploadOutput with error: %v.", err)
				output = nil
			}
		}
	}
	return
}

// UploadPart uploads a part to a specified bucket by using a specified multipart upload ID.
//
// After a multipart upload is initialized, you can use this API to upload a part to a specified bucket
// by using the multipart upload ID. Except for the last uploaded part whose size ranges from 0 to 5 GB,
// sizes of the other parts range from 100 KB to 5 GB. The upload part ID ranges from 1 to 10000.
func (wosClient WosClient) UploadPart(_input *UploadPartInput, extensions ...extensionOptions) (output *UploadPartOutput, err error) {
	if _input == nil {
		return nil, errors.New("UploadPartInput is nil")
	}

	if _input.UploadId == "" {
		return nil, errors.New("UploadId is empty")
	}

	input := &UploadPartInput{}
	input.Bucket = _input.Bucket
	input.Key = _input.Key
	input.PartNumber = _input.PartNumber
	input.UploadId = _input.UploadId
	input.ContentMD5 = _input.ContentMD5
	input.SourceFile = _input.SourceFile
	input.Offset = _input.Offset
	input.PartSize = _input.PartSize
	input.Body = _input.Body

	output = &UploadPartOutput{}
	var repeatable bool
	if input.Body != nil {
		_, repeatable = input.Body.(*strings.Reader)
		if _, ok := input.Body.(*readerWrapper); !ok && input.PartSize > 0 {
			input.Body = &readerWrapper{reader: input.Body, totalCount: input.PartSize}
		}
	} else if sourceFile := strings.TrimSpace(input.SourceFile); sourceFile != "" {
		fd, _err := os.Open(sourceFile)
		if _err != nil {
			err = _err
			return nil, err
		}
		defer func() {
			errMsg := fd.Close()
			if errMsg != nil {
				doLog(LEVEL_WARN, "Failed to close file with reason: %v", errMsg)
			}
		}()

		stat, _err := fd.Stat()
		if _err != nil {
			err = _err
			return nil, err
		}
		fileSize := stat.Size()
		fileReaderWrapper := &fileReaderWrapper{filePath: sourceFile}
		fileReaderWrapper.reader = fd

		if input.Offset < 0 || input.Offset > fileSize {
			input.Offset = 0
		}

		if input.PartSize <= 0 || input.PartSize > (fileSize-input.Offset) {
			input.PartSize = fileSize - input.Offset
		}
		fileReaderWrapper.totalCount = input.PartSize
		fileReaderWrapper.mark = input.Offset
		if _, err = fd.Seek(input.Offset, io.SeekStart); err != nil {
			return nil, err
		}
		input.Body = fileReaderWrapper
		repeatable = true
	}
	if repeatable {
		err = wosClient.doActionWithBucketAndKey("UploadPart", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	} else {
		err = wosClient.doActionWithBucketAndKeyUnRepeatable("UploadPart", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	}
	if err != nil {
		output = nil
	} else {
		ParseUploadPartOutput(output)
		output.PartNumber = input.PartNumber
	}
	return
}

// CompleteMultipartUpload combines the uploaded parts in a specified bucket by using the multipart upload ID.
func (wosClient WosClient) CompleteMultipartUpload(input *CompleteMultipartUploadInput, extensions ...extensionOptions) (output *CompleteMultipartUploadOutput, err error) {
	if input == nil {
		return nil, errors.New("CompleteMultipartUploadInput is nil")
	}

	if input.UploadId == "" {
		return nil, errors.New("UploadId is empty")
	}

	var parts partSlice = input.Parts
	sort.Sort(parts)

	output = &CompleteMultipartUploadOutput{}
	err = wosClient.doActionWithBucketAndKey("CompleteMultipartUpload", HTTP_POST, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseCompleteMultipartUploadOutput(output)
		if output.EncodingType == "url" {
			err = decodeCompleteMultipartUploadOutput(output)
			if err != nil {
				doLog(LEVEL_ERROR, "Failed to get CompleteMultipartUploadOutput with error: %v.", err)
				output = nil
			}
		}
	}
	return
}

// ListParts lists the uploaded parts in a bucket by using the multipart upload ID.
func (wosClient WosClient) ListParts(input *ListPartsInput, extensions ...extensionOptions) (output *ListPartsOutput, err error) {
	if input == nil {
		return nil, errors.New("ListPartsInput is nil")
	}
	if input.UploadId == "" {
		return nil, errors.New("UploadId is empty")
	}
	output = &ListPartsOutput{}
	err = wosClient.doActionWithBucketAndKey("ListParts", HTTP_GET, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else if output.EncodingType == "url" {
		err = decodeListPartsOutput(output)
		if err != nil {
			doLog(LEVEL_ERROR, "Failed to get ListPartsOutput with error: %v.", err)
			output = nil
		}
	}
	return
}

// CopyPart copy a part to a specified bucket by using a specified multipart upload ID.
//
// After a multipart upload is initialized, you can use this API to copy a part to a specified bucket by using the multipart upload ID.
func (wosClient WosClient) CopyPart(input *CopyPartInput, extensions ...extensionOptions) (output *CopyPartOutput, err error) {
	if input == nil {
		return nil, errors.New("CopyPartInput is nil")
	}
	if input.UploadId == "" {
		return nil, errors.New("UploadId is empty")
	}
	if strings.TrimSpace(input.CopySourceBucket) == "" {
		return nil, errors.New("Source bucket is empty")
	}
	if strings.TrimSpace(input.CopySourceKey) == "" {
		return nil, errors.New("Source key is empty")
	}

	output = &CopyPartOutput{}
	err = wosClient.doActionWithBucketAndKey("CopyPart", HTTP_PUT, input.Bucket, input.Key, input, output, extensions)
	if err != nil {
		output = nil
	} else {
		ParseCopyPartOutput(output)
		output.PartNumber = input.PartNumber
	}
	return
}

// UploadFile resume uploads.
//
// This API is an encapsulated and enhanced version of multipart upload, and aims to eliminate large file
// upload failures caused by poor network conditions and program breakdowns.
func (wosClient WosClient) UploadFile(input *UploadFileInput, extensions ...extensionOptions) (output *CompleteMultipartUploadOutput, err error) {
	if input.EnableCheckpoint && input.CheckpointFile == "" {
		input.CheckpointFile = input.UploadFile + ".uploadfile_record"
	}

	if input.TaskNum <= 0 {
		input.TaskNum = 1
	}
	if input.PartSize < MIN_PART_SIZE {
		input.PartSize = MIN_PART_SIZE
	} else if input.PartSize > MAX_PART_SIZE {
		input.PartSize = MAX_PART_SIZE
	}

	output, err = wosClient.resumeUpload(input, extensions)
	return
}

// DownloadFile resume downloads.
//
// This API is an encapsulated and enhanced version of partial download, and aims to eliminate large file
// download failures caused by poor network conditions and program breakdowns.
func (wosClient WosClient) DownloadFile(input *DownloadFileInput, extensions ...extensionOptions) (output *GetObjectMetadataOutput, err error) {
	if input.DownloadFile == "" {
		input.DownloadFile = input.Key
	}

	if input.EnableCheckpoint && input.CheckpointFile == "" {
		input.CheckpointFile = input.DownloadFile + ".downloadfile_record"
	}

	if input.TaskNum <= 0 {
		input.TaskNum = 1
	}
	if input.PartSize <= 0 {
		input.PartSize = DEFAULT_PART_SIZE
	}

	output, err = wosClient.resumeDownload(input, extensions)
	return
}

// SetBucketFetchPolicy sets the bucket fetch policy.
//
// You can use this API to set a bucket fetch policy.
func (wosClient WosClient) SetBucketFetchPolicy(input *SetBucketFetchPolicyInput, extensions ...extensionOptions) (output *BaseModel, err error) {
	if input == nil {
		return nil, errors.New("SetBucketFetchPolicyInput is nil")
	}
	if strings.TrimSpace(string(input.Status)) == "" {
		return nil, errors.New("Fetch policy status is empty")
	}
	if strings.TrimSpace(input.Agency) == "" {
		return nil, errors.New("Fetch policy agency is empty")
	}
	output = &BaseModel{}
	err = wosClient.doActionWithBucketAndKey("SetBucketFetchPolicy", HTTP_PUT, input.Bucket, string(objectKeyExtensionPolicy), input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// GetBucketFetchPolicy gets the bucket fetch policy.
//
// You can use this API to obtain the fetch policy of a bucket.
func (wosClient WosClient) GetBucketFetchPolicy(input *GetBucketFetchPolicyInput, extensions ...extensionOptions) (output *GetBucketFetchPolicyOutput, err error) {
	if input == nil {
		return nil, errors.New("GetBucketFetchPolicyInput is nil")
	}
	output = &GetBucketFetchPolicyOutput{}
	err = wosClient.doActionWithBucketAndKeyV2("GetBucketFetchPolicy", HTTP_GET, input.Bucket, string(objectKeyExtensionPolicy), input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// DeleteBucketFetchPolicy deletes the bucket fetch policy.
//
// You can use this API to delete the fetch policy of a bucket.
func (wosClient WosClient) DeleteBucketFetchPolicy(input *DeleteBucketFetchPolicyInput, extensions ...extensionOptions) (output *BaseModel, err error) {
	if input == nil {
		return nil, errors.New("DeleteBucketFetchPolicyInput is nil")
	}
	output = &BaseModel{}
	err = wosClient.doActionWithBucketAndKey("DeleteBucketFetchPolicy", HTTP_DELETE, input.Bucket, string(objectKeyExtensionPolicy), input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// SetBucketFetchJob sets the bucket fetch job.
//
// You can use this API to set a bucket fetch job.
func (wosClient WosClient) SetBucketFetchJob(input *SetBucketFetchJobInput, extensions ...extensionOptions) (output *SetBucketFetchJobOutput, err error) {
	if input == nil {
		return nil, errors.New("SetBucketFetchJobInput is nil")
	}
	if strings.TrimSpace(input.URL) == "" {
		return nil, errors.New("URL is empty")
	}
	output = &SetBucketFetchJobOutput{}
	err = wosClient.doActionWithBucketAndKeyV2("SetBucketFetchJob", HTTP_POST, input.Bucket, string(objectKeyAsyncFetchJob), input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}

// GetBucketFetchJob gets the bucket fetch job.
//
// You can use this API to obtain the fetch job of a bucket.
func (wosClient WosClient) GetBucketFetchJob(input *GetBucketFetchJobInput, extensions ...extensionOptions) (output *GetBucketFetchJobOutput, err error) {
	if input == nil {
		return nil, errors.New("GetBucketFetchJobInput is nil")
	}
	if strings.TrimSpace(input.JobID) == "" {
		return nil, errors.New("JobID is empty")
	}
	output = &GetBucketFetchJobOutput{}
	err = wosClient.doActionWithBucketAndKeyV2("GetBucketFetchJob", HTTP_GET, input.Bucket, string(objectKeyAsyncFetchJob)+"/"+input.JobID, input, output, extensions)
	if err != nil {
		output = nil
	}
	return
}
