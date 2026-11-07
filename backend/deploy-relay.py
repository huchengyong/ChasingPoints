"""生产发布包的七牛中转：在生产服务器上执行（deploy-prod.sh 通过 ssh 标准输入发送本脚本）。

七牛密钥只从服务器 .env 读取，不离开服务器：
  upload-token BUCKET KEY          输出仅能写入 BUCKET:KEY、1 小时后过期的上传凭证
  fetch BUCKET KEY DEST SHA256     经 S3 兼容接口下载到 DEST 并校验 sha256，随后删除七牛上的对象

只用标准库，兼容服务器上的 Python 3.6。空间需与 S3_REGION 同区域（华南 z2）。
"""
import base64
import datetime
import hashlib
import hmac
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

ENV_FILE = "/www/wwwroot/api.zhuifen.cn/.env"
S3_REGION = "cn-south-1"
S3_HOST = "s3.cn-south-1.qiniucs.com"
TOKEN_TTL_SECONDS = 3600
MAX_ARCHIVE_BYTES = 200 * 1024 * 1024


def die(message):
    sys.stderr.write(message + "\n")
    sys.exit(1)


def load_keys():
    values = {}
    with open(ENV_FILE) as env:
        for line in env:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            name, value = line.split("=", 1)
            values[name.strip()] = value.strip().strip("\"'")
    access_key = values.get("QINIU_ACCESS_KEY")
    secret_key = values.get("QINIU_SECRET_KEY")
    if not access_key or not secret_key:
        die("QINIU_ACCESS_KEY / QINIU_SECRET_KEY missing from " + ENV_FILE)
    return access_key, secret_key


def upload_token(access_key, secret_key, bucket, key):
    policy = {
        "scope": "%s:%s" % (bucket, key),
        "deadline": int(time.time()) + TOKEN_TTL_SECONDS,
        "fsizeLimit": MAX_ARCHIVE_BYTES,
    }
    encoded_policy = base64.urlsafe_b64encode(json.dumps(policy, separators=(",", ":")).encode())
    signature = base64.urlsafe_b64encode(hmac.new(secret_key.encode(), encoded_policy, hashlib.sha1).digest())
    return "%s:%s:%s" % (access_key, signature.decode(), encoded_policy.decode())


def s3_request(access_key, secret_key, method, bucket, key):
    """按 AWS Signature V4 签名访问七牛 S3 兼容接口（path-style）。"""
    now = datetime.datetime.utcnow()
    amz_date = now.strftime("%Y%m%dT%H%M%SZ")
    date = now.strftime("%Y%m%d")
    uri = "/" + bucket + "/" + urllib.parse.quote(key, safe="/~")
    payload_hash = hashlib.sha256(b"").hexdigest()
    signed_headers = "host;x-amz-content-sha256;x-amz-date"
    canonical_request = "\n".join([
        method,
        uri,
        "",
        "host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n" % (S3_HOST, payload_hash, amz_date),
        signed_headers,
        payload_hash,
    ])
    scope = "%s/%s/s3/aws4_request" % (date, S3_REGION)
    string_to_sign = "\n".join([
        "AWS4-HMAC-SHA256",
        amz_date,
        scope,
        hashlib.sha256(canonical_request.encode()).hexdigest(),
    ])
    signing_key = ("AWS4" + secret_key).encode()
    for part in (date, S3_REGION, "s3", "aws4_request"):
        signing_key = hmac.new(signing_key, part.encode(), hashlib.sha256).digest()
    signature = hmac.new(signing_key, string_to_sign.encode(), hashlib.sha256).hexdigest()
    request = urllib.request.Request("https://" + S3_HOST + uri, method=method, headers={
        "x-amz-date": amz_date,
        "x-amz-content-sha256": payload_hash,
        "Authorization": "AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s"
        % (access_key, scope, signed_headers, signature),
    })
    return urllib.request.urlopen(request, timeout=120)


def delete_object(access_key, secret_key, bucket, key):
    try:
        s3_request(access_key, secret_key, "DELETE", bucket, key).close()
    except (urllib.error.URLError, OSError) as error:
        sys.stderr.write("warning: failed to delete relay object %s/%s: %s\n" % (bucket, key, error))


def fetch(access_key, secret_key, bucket, key, dest, expected_sha256):
    started = time.time()
    digest = hashlib.sha256()
    size = 0
    try:
        with s3_request(access_key, secret_key, "GET", bucket, key) as response, open(dest, "wb") as out:
            while True:
                chunk = response.read(1024 * 1024)
                if not chunk:
                    break
                digest.update(chunk)
                out.write(chunk)
                size += len(chunk)
        if digest.hexdigest() != expected_sha256:
            os.remove(dest)
            die("relay archive checksum mismatch: expected %s, got %s" % (expected_sha256, digest.hexdigest()))
    except (urllib.error.URLError, OSError) as error:
        if os.path.exists(dest):
            os.remove(dest)
        die("failed to download relay object %s/%s: %s" % (bucket, key, error))
    finally:
        delete_object(access_key, secret_key, bucket, key)
    print("server downloaded %d bytes from relay in %.1fs" % (size, time.time() - started))


def main(argv):
    if len(argv) == 3 and argv[0] == "upload-token":
        access_key, secret_key = load_keys()
        print(upload_token(access_key, secret_key, argv[1], argv[2]))
    elif len(argv) == 5 and argv[0] == "fetch":
        access_key, secret_key = load_keys()
        fetch(access_key, secret_key, argv[1], argv[2], argv[3], argv[4])
    else:
        die(__doc__)


if __name__ == "__main__":
    main(sys.argv[1:])
