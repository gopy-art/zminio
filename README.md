<div align="center">

[![GitHub go.mod Go version of a Go module](https://img.shields.io/badge/go-1.23.2-blue)](https://go.dev/dl/) 
[![GitHub go.mod Go version of a Go module](https://img.shields.io/badge/wotk_with-prometheus-red)](https://go.dev/dl/)
[![GitHub go.mod Go version of a Go module](https://img.shields.io/badge/work_with-minio-orange)](https://go.dev/dl/)
[![GitHub go.mod Go version of a Go module](https://img.shields.io/badge/app_version-2.2.0-green)](https://go.dev/dl/)
</div>

# zminio

this is a minio client package that allows you to these actions with your minio server.

1 ) `upload` : upload files to your minio server. ( every file you want! ) <br>
2 ) `uploadDir` : upload a directory full of multiple files to the minio. <br>
3 ) `download` : download one object or all of the objects from one bucket in minio server. <br>
4 ) `delete` : delete one object or all of the objects from one bucket in minio server. <br>
5 ) `list` : get the list of objects from one bucket in minio server. <br>
6 ) `info` : get the information of one object from minio server. <br>
7 ) `move` : move one object from one bucket to another bucket in minio server. <br>
8 ) `sync` : sync two bucket from minio server with each other. <br>
9 ) `listenDownload` : listen to the one bucket in minio server , and download every object that will be uploaded.

> [!NOTE] 
> you can set the config of the minio servers from flags and .env file.

> [!NOTE] 
> This app is suppporting concurrency, and you can set the amount of workers with `-n` flag. (default = 10)

### Flags
- ak : set your minio access key
- aks : set your minio access key
- b : set your minio bucket name
- bs : set your minio bucket name
- do : set the job you want to do. (download, upload, move, delete, list, sync, uploadDir)
- ds : delete the object from bucket after sync
- f : set the path of the file that you wanna upload
- l : set app logger type , stdout or file (default "stdout")
- mb : set your minio bucket name that you want to move files to it
- n : set the count of worker for run (default 10)
- o : set the path of the file that you wanna download
- obj : set your minio object name
- pr : run Prometheus on ip:port to monitor aminio metrics,if not set this flag prometheus disabled. (exaple:-pr 0.0.0.0:1234)
- se : set your secure ssl option in connecting to the minio (default true)
- ses : set your secure ssl option in connecting to the minio (default true)
- sk : set your minio secret key
- sks : set your minio secret key
- u : set your minio url address
- us : set your minio url address
- v : zminio version

## Sample Commands
<strong> NOTE : </strong> you have to create .env file for set minio login data or you could pass them via the provided flags.

- upload :
```
./Zminio -f README.md -do upload
```

- download :
```
sudo ./Zminio -obj README.md -o /tmp -do download
```

- list :
```
./Zminio -do list
```

## ENV file
the format of .env file should be like this:
```
MINIO_ENDPOINT="localhost:9000"
MINIO_ACCESS_KEY="minioadmin"
MINIO_SECRET_KEY="minioadmin"
MINIO_BUCKET_NAME="sync"
MINIO_SSL_SECRET="false"
SYNC_MINIO_ENDPOINT="localhost:8000"
SYNC_MINIO_ACCESS_KEY="minioadmin"
SYNC_MINIO_SECRET_KEY="minioadmin"
SYNC_MINIO_BUCKET_NAME="upload"
SYNC_MINIO_SSL_SECRET="false"
```