# arti-ssh Proxy Agent

**arti-ssh-proxy**는 Go 언어로 작성된 경량 서버 에이전트입니다.  
`your-domain.com` 웹 허브와 보안 WebSocket(WSS)으로 연결되어, 웹 브라우저에서 대상 서버의 쉘(Bash/Zsh)에 직접 접근할 수 있게 해줍니다.

## 📌 아키텍처 (Architecture)

```mermaid
graph LR
    User[Web Client] --HTTPS--> Hub[your-domain.com]
    Hub --WSS (Secure WebSocket)--> Agent[arti-ssh-proxy]
    Agent --PTY (Local Shell)--> Shell[Bash/Zsh]
```

- **Web Client**: 사용자가 브라우저를 통해 접속합니다.
- **Hub (Server)**: 여러 에이전트와의 연결을 관리하고 클라이언트와 중계합니다.
- **Agent**: 각 대상 서버에 설치되어 실제 터미널 명령을 수행하고 결과를 허브로 전송합니다.

## 🚀 기능 (Features)

- **보안 통신**: TLS 1.3 기반의 WSS 터널링.
- **PTY 지원**: `creack/pty`를 이용한 완전한 터미널 에뮬레이션 지원 (Vim, Top 등 사용 가능).
- **자동 재연결**: 네트워크 단절 시 자동으로 재연결을 시도하여 가용성 확보.
- **단일 바이너리**: 외부 라이브러리 의존성 없는 정적 바이너리로 배포 용이.
- **크로스 컴파일**: Linux x86_64 및 ARM64 지원.

## 🛠 빌드 방법 (Build)

Go 1.23 이상이 필요합니다. 포함된 `build.sh`를 사용하거나 직접 빌드할 수 있습니다.

### 자동 빌드 스크립트 사용
```bash
chmod +x build.sh
./build.sh
```
`bin/` 디렉토리에 `arti-ssh-agent-amd64` (x86_64)와 `arti-ssh-agent-arm64` (ARM)가 생성됩니다.

### 수동 빌드
```bash
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o arti-ssh-agent ./cmd/agent

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o arti-ssh-agent ./cmd/agent
```

## 🔄 CI/CD 및 릴리스 (Release)

GitHub Actions를 통해 태그가 푸시될 때 자동으로 빌드 및 릴리스가 생성됩니다.

### 릴리스 절차

1. 코드를 커밋하고 푸시합니다.
2. `v`로 시작하는 태그를 생성하고 푸시합니다.
   ```bash
   git tag v0.0.1
   git push origin v0.0.1
   ```
3. GitHub Actions가 자동으로 다음 작업을 수행합니다:
   - Linux x86_64 빌드
   - Linux ARM64 빌드
   - GitHub Releases 페이지에 바이너리 업로드

## ⚡ 자동 설치 (Automated Install)

`curl`을 사용하여 최신 버전을 자동으로 설치할 수 있습니다.

```bash
# 최신 버전 설치
curl -sL https://raw.githubusercontent.com/SaAkSin/arti-ssh-proxy/main/install.sh | sudo bash

# 특정 버전 설치 (예: v0.0.1)
curl -sL https://raw.githubusercontent.com/SaAkSin/arti-ssh-proxy/main/install.sh | sudo bash -s v0.0.1
```

---

## ☁️ 배포 및 설정 가이드 (Amazon Linux 2023 / Rocky Linux 9)

**Amazon Linux 2023**과 **Rocky Linux 9**(RHEL 9 호환)은 모두 **Systemd**를 사용하므로 설정 방법이 거의 동일합니다.
주요 차이점은 기본 사용자 계정명입니다 (AWS 기준: Amazon Linux=`ec2-user`, Rocky Linux=`rocky`).

### 1. 바이너리 설치

서버에 빌드된 바이너리를 업로드하고 실행 권한을 부여합니다.

```bash
# 예: /usr/local/bin 에 설치
sudo mv arti-ssh-agent-amd64 /usr/local/bin/arti-ssh-agent
sudo chmod +x /usr/local/bin/arti-ssh-agent
```

### 2. Systemd 서비스 파일 생성

`/etc/systemd/system/arti-ssh.service` 파일을 생성합니다.

```bash
sudo vi /etc/systemd/system/arti-ssh.service
```

다음 내용을 붙여넣으세요.
> **주의**: `User` 항목을 사용 중인 배포판에 맞게 수정해야 합니다.
> - **Amazon Linux 2023**: `User=ec2-user`
> - **Rocky Linux 9**: `User=rocky`

```ini
[Unit]
Description=Arti SSH Proxy Agent
After=network.target

[Service]
Type=simple
# 배포판에 맞는 사용자로 변경하세요 (ec2-user 또는 rocky)
User=ec2-user
Group=ec2-user

# 실제 연결할 웹소켓 URL로 수정하세요. 
# 예: 토큰이 필요한 경우 ?token=XYZ 추가
ExecStart=/usr/local/bin/arti-ssh-agent -url wss://your-domain.com/ws

# 비정상 종료 시 자동 재시작
Restart=always
RestartSec=5

# 로그 출력 설정
StandardOutput=journal
StandardError=journal

# 환경 변수로 설정 (Optional)
# ARTI_SSH_URL 환경 변수를 통해 URL을 설정할 수도 있습니다.
# Environment=ARTI_SSH_URL=wss://your-domain.com/ws

[Install]
WantedBy=multi-user.target
```

### 3. 서비스 시작 및 활성화

설정을 반영하고 서비스를 시작합니다.

```bash
# Systemd 데몬 리로드
sudo systemctl daemon-reload

# 서비스 시작
sudo systemctl start arti-ssh

# 부팅 시 자동 시작 설정
sudo systemctl enable arti-ssh
```

### 4. 상태 및 로그 확인

서비스가 정상적으로 실행 중인지 확인합니다.

```bash
# 상태 확인
sudo systemctl status arti-ssh

# 로그 실시간 확인
sudo journalctl -u arti-ssh -f
```

## 📝 라이선스
MIT License
