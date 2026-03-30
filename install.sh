#!/bin/bash

# ==========================================================
# X-Panel 统一安装脚本 (付费/免费二合一)
# 作者: X-Panel
# ==========================================================

red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

# check root
[[ $EUID -ne 0 ]] && echo -e "${red}致命错误: ${plain} 请使用 root 权限运行此脚本\n" && exit 1

get_release_tag() {
    local mode="${1:-stable}"
    if [[ "${mode}" == "pre" ]]; then
        curl -Ls "https://api.github.com/repos/Sora39831/X-Panel/releases?per_page=30"         | awk 'BEGIN{RS="},"} /"prerelease":[[:space:]]*true/ { if (match($0, /"tag_name":[[:space:]]*"([^"]+)"/, m)) { print m[1]; exit } }'
    else
        curl -Ls "https://api.github.com/repos/Sora39831/X-Panel/releases/latest"         | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    fi
}

choose_install_release_tag() {
    local stable_version pre_version version_choice
    stable_version=$(get_release_tag stable)
    pre_version=$(get_release_tag pre)

    if [[ -z "$stable_version" && -z "$pre_version" ]]; then
        echo "" >&2
        echo -e "${red}获取 X-Panel 版本失败，可能是 Github API 限制，请稍后再试${plain}" >&2
        exit 1
    fi

    if [[ -n "$pre_version" ]]; then
        echo "" >&2
        echo -e "${yellow}检测到最新 pre-release：${pre_version}${plain}" >&2
        echo -e "${yellow}检测到最新稳定版：${stable_version}${plain}" >&2
        echo "" >&2
        echo -e "  ${green}1)${plain} 安装最新稳定版 ${yellow}${stable_version}${plain}" >&2
        echo -e "  ${green}2)${plain} 安装最新 pre-release ${yellow}${pre_version}${plain}" >&2
        read -p "请选择要安装的版本 (1 或 2，默认 1): " version_choice < /dev/tty

        case "$version_choice" in
            2) echo "$pre_version" ;;
            *) echo "$stable_version" ;;
        esac
    else
        echo "$stable_version"
    fi
}

# ----------------------------------------------------------
# 获取机器唯一硬件标识 (HWID)
# ----------------------------------------------------------
get_hwid() {
    local machine_id=""

    # 1. 优先尝试获取 DMI Product UUID (VPS 硬件 ID，重装系统通常不变)
    if [[ -r /sys/class/dmi/id/product_uuid ]]; then
        machine_id=$(cat /sys/class/dmi/id/product_uuid)
    
    # 2. 其次尝试获取 eth0 网卡 MAC 地址 (大部分 VPS 重装后 MAC 不变)
    elif [[ -r /sys/class/net/eth0/address ]]; then
        machine_id=$(cat /sys/class/net/eth0/address)
        
    # 3. 如果都失败，才使用 machine-id (重装会变，作为最后兜底)
    elif [[ -f /etc/machine-id ]]; then
        machine_id=$(cat /etc/machine-id)
    else
        machine_id=$(hostname)
    fi
    
    # 取 MD5 作为唯一指纹，确保格式统一
    echo -n "$machine_id" | md5sum | awk '{print $1}'
}

# ----------------------------------------------------------
# 函数：付费Pro版安装逻辑 (install_paid_version)
# ----------------------------------------------------------
# 此函数负责获取授权码和IP + 机器指纹，并从远程授权服务器获取并执行付费脚本
#
install_paid_version() {
    echo ""
    echo -e "${green}您正在安装/升级/更新 【X-Panel 付费Pro版】${plain}"
    echo ""
    echo -e "${yellow}------------------------------------------------------${plain}"
    echo ""

    # 1. 提示用户输入授权码
    read -p "$(echo -e "${yellow}请输入您的授权码 (License Key): ${plain}")" auth_key
    echo ""
    
    if [ -z "$auth_key" ]; then
        echo -e "${red}错误: 您没有输入授权码。${plain}"
        exit 1
    fi
    
    # 2. 获取本机的公共 IPv4 地址
    echo -e "${green}正在获取本机 IP 地址......${plain}"
    vps_ip=$(curl -s4m8 ip.sb -k | head -n 1)
    
    if [ -z "$vps_ip" ]; then
        echo -e "${red}致命错误: 未能获取服务器的公共 IP 地址。${plain}"
        echo -e "${red}请检查您的网络连接或 curl 是否正常工作。${plain}"
        exit 1
    fi

    # 3. [新增] 获取本机硬件指纹
    vps_hwid=$(get_hwid)

    echo -e "${green}本机 IP: ${vps_ip}${plain}"
    echo -e "${green}机器指纹: ${vps_hwid}${plain}" # 调试用
    echo ""
    
    # 4. 设置您的授权服务器地址
    AUTH_SERVER_URL="https://auth.x-panel.vip/install_pro.php"
    
    echo -e "${green}正在连接〔远程授权服务器〕进行验证......${plain}"
    echo ""
    echo -e "${yellow}请稍候.........${plain}"
    
    # 5. 将服务器响应保存到变量
    response=$(curl -sL --connect-timeout 20 -X POST -d "key=${auth_key}&ip=${vps_ip}&hwid=${vps_hwid}" "${AUTH_SERVER_URL}")
    
    # 6. 简单判断响应是否为空
    if [ -z "$response" ]; then
        echo -e "${red}错误: 无法连接到授权服务器或服务器无响应。${plain}"
        echo -e "${yellow}请检查网络连接或联系管理员。${plain}"
        exit 1
    fi

    # 7. 判断是否包含 PHP 错误 (如 Syntax error 或 Fatal error)
    # 如果 PHP 报错，通常会包含 "Fatal error" 或 "Parse error" 字样
    if echo "$response" | grep -qE "Fatal error|Parse error"; then
         echo -e "${red}错误: 授权服务器发生内部错误。${plain}"
         echo -e "详细信息: $response"
         exit 1
    fi

    # 8. 执行脚本
    bash <(echo "$response")
    
    exit 0
}


# ----------------------------------------------------------
# 函数：免费基础版安装逻辑 (install_free_version) 
# ----------------------------------------------------------
install_free_version() {
    echo ""
    echo -e "${green}您选择了安装 【X-Panel 免费基础版】${plain}"
    echo ""
    echo -e "${green}即将开始执行标准安装流程...${plain}"
    sleep 2

    cur_dir=$(pwd)

    # Check OS and set release variable
    if [[ -f /etc/os-release ]]; then
        source /etc/os-release
        release=$ID
    elif [[ -f /usr/lib/os-release ]]; then
        source /usr/lib/os-release
        release=$ID
    else
        echo "" >&2
        echo -e "${red}检查服务器操作系统失败，请联系作者!${plain}" >&2
        exit 1
    fi
    echo ""
    echo -e "${green}---------->>>>>目前服务器的操作系统为: $release${plain}"

    arch() {
        case "$(uname -m)" in
            x86_64 | x64 | amd64 ) echo 'amd64' ;;
            i*86 | x86 ) echo '386' ;;
            armv8* | armv8 | arm64 | aarch64 ) echo 'arm64' ;;
            armv7* | armv7 | arm ) echo 'armv7' ;;
            armv6* | armv6 ) echo 'armv6' ;;
            armv5* | armv5 ) echo 'armv5' ;;
            s390x) echo 's390x' ;;
            *) echo -e "${green}不支持的CPU架构! ${plain}" && rm -f install.sh && exit 1 ;;
        esac
    }

    echo ""
    # check_glibc_version() {
    #    glibc_version=$(ldd --version | head -n1 | awk '{print $NF}')

    #    required_version="2.32"
    #    if [[ "$(printf '%s\n' "$required_version" "$glibc_version" | sort -V | head -n1)" != "$required_version" ]]; then
    #        echo -e "${red}------>>>GLIBC版本 $glibc_version 太旧了！ 要求2.32或以上版本${plain}"
    #        echo -e "${green}-------->>>>请升级到较新版本的操作系统以便获取更高版本的GLIBC${plain}"
    #        exit 1
    #    fi
    #        echo -e "${green}-------->>>>GLIBC版本： $glibc_version（符合高于2.32的要求）${plain}"
    # }
    # check_glibc_version

    # echo ""
    echo -e "${yellow}---------->>>>>当前系统的架构为: $(arch)${plain}"
    echo ""
    last_version=$(get_release_tag stable)
    # 获取 x-ui 版本
    xui_version=$(/usr/local/x-ui/x-ui -v 2>/dev/null)

    # 检查 xui_version 是否为空
    if [[ -z "$xui_version" ]]; then
        echo "" >&2
        echo -e "${red}------>>>当前服务器没有安装任何 x-ui 系列代理面板${plain}"
        echo "" >&2
        echo -e "${green}-------->>>>片刻之后脚本将会自动引导安装〔X-Panel面板〕${plain}"
    else
        # 检查版本号中是否包含冒号
        if [[ "$xui_version" == *:* ]]; then
            echo -e "${green}---------->>>>>当前代理面板的版本为: ${red}其他 x-ui 分支版本${plain}"
            echo "" >&2
            echo -e "${green}-------->>>>片刻之后脚本将会自动引导安装〔X-Panel面板〕${plain}"
        else
            echo -e "${green}---------->>>>>当前代理面板的版本为: ${red}〔X-Panel面板〕v${xui_version}${plain}"
        fi
    fi
    echo ""
    echo -e "${yellow}---------------------->>>>>〔X-Panel面板〕最新版为：${last_version}${plain}"
    sleep 4

    os_version=$(grep -i version_id /etc/os-release | cut -d \" -f2 | cut -d . -f1)

    if [[ "${release}" == "arch" ]]; then
        echo "您的操作系统是 ArchLinux"
    elif [[ "${release}" == "manjaro" ]]; then
        echo "您的操作系统是 Manjaro"
    elif [[ "${release}" == "armbian" ]]; then
        echo "您的操作系统是 Armbian"
    elif [[ "${release}" == "alpine" ]]; then
        echo "您的操作系统是 Alpine Linux"
    elif [[ "${release}" == "opensuse-tumbleweed" ]]; then
        echo "您的操作系统是 OpenSUSE Tumbleweed"
    elif [[ "${release}" == "centos" ]]; then
        if [[ ${os_version} -lt 8 ]]; then
            echo -e "${red} 请使用 CentOS 8 或更高版本 ${plain}\n" && exit 1
        fi
    elif [[ "${release}" == "ubuntu" ]]; then
        if [[ ${os_version} -lt 20 ]]; then
            echo -e "${red} 请使用 Ubuntu 20 或更高版本!${plain}\n" && exit 1
        fi
    elif [[ "${release}" == "fedora" ]]; then
        if [[ ${os_version} -lt 36 ]]; then
            echo -e "${red} 请使用 Fedora 36 或更高版本!${plain}\n" && exit 1
        fi
    elif [[ "${release}" == "debian" ]]; then
        if [[ ${os_version} -lt 11 ]]; then
            echo -e "${red} 请使用 Debian 11 或更高版本 ${plain}\n" && exit 1
        fi
    elif [[ "${release}" == "almalinux" ]]; then
        if [[ ${os_version} -lt 9 ]]; then
            echo -e "${red} 请使用 AlmaLinux 9 或更高版本 ${plain}\n" && exit 1
        fi
    elif [[ "${release}" == "rocky" ]]; then
        if [[ ${os_version} -lt 9 ]]; then
            echo -e "${red} 请使用 RockyLinux 9 或更高版本 ${plain}\n" && exit 1
        fi
    elif [[ "${release}" == "oracle" ]]; then
        if [[ ${os_version} -lt 8 ]]; then
            echo -e "${red} 请使用 Oracle Linux 8 或更高版本 ${plain}\n" && exit 1
        fi
    else
        echo -e "${red}此脚本不支持您的操作系统。${plain}\n"
        echo "请确保您使用的是以下受支持的操作系统之一："
        echo "- Ubuntu 20.04+"
        echo "- Debian 11+"
        echo "- CentOS 8+"
        echo "- Fedora 36+"
        echo "- Arch Linux"
        echo "- Manjaro"
        echo "- Armbian"
        echo "- Alpine Linux"
        echo "- AlmaLinux 9+"
        echo "- Rocky Linux 9+"
        echo "- Oracle Linux 8+"
        echo "- OpenSUSE Tumbleweed"
        exit 1

    fi

    install_base() {
        case "${release}" in
        ubuntu | debian | armbian)
            apt-get update && apt-get install -y -q wget curl sudo tar tzdata
            ;;
        centos | rhel | almalinux | rocky | ol)
            yum -y --exclude=kernel* update && yum install -y -q wget curl sudo tar tzdata
            ;;
        fedora | amzn | virtuozzo)
            dnf -y --exclude=kernel* update && dnf install -y -q wget curl sudo tar tzdata
            ;;
        arch | manjaro | parch)
            pacman -Sy && pacman -S --noconfirm wget curl sudo tar tzdata
            ;;
        alpine)
            apk update && apk add --no-cache wget curl sudo tar tzdata
            ;;
        opensuse-tumbleweed)
            zypper refresh && zypper -q install -y wget curl sudo tar timezone
            ;;
        *)
            apt-get update && apt-get install -y -q wget curl sudo tar tzdata
            ;;
        esac
    }

    choose_install_db_type() {
        local db_type_conf="/etc/x-ui/db-type.conf"
        mkdir -p /etc/x-ui
        if [[ -f "${db_type_conf}" ]]; then
            source "${db_type_conf}"
            XUI_DB_TYPE=${XUI_DB_TYPE:-sqlite}
            echo -e "${green}检测到现有数据库配置，继续使用: ${yellow}${XUI_DB_TYPE}${plain}"
        else
            echo -e "${green}请选择数据库类型:${plain}"
            echo -e "  ${green}1)${plain} SQLite (默认)"
            echo -e "  ${green}2)${plain} MongoDB"
            read -p "请输入选择 [1-2]，直接回车默认 SQLite: " db_choice
            case "${db_choice}" in
                2)
                    XUI_DB_TYPE="mongodb"
                    ;;
                *)
                    XUI_DB_TYPE="sqlite"
                    ;;
            esac
            cat >"${db_type_conf}" <<EOF
XUI_DB_TYPE=${XUI_DB_TYPE}
EOF
            echo -e "${green}数据库类型已设置为: ${yellow}${XUI_DB_TYPE}${plain}"
        fi
        INSTALL_DB_TYPE="${XUI_DB_TYPE}"
        export XUI_DB_TYPE="${INSTALL_DB_TYPE}"
    }

    install_mongodb_runtime_noninteractive() {
        echo -e "${green}正在安装 MongoDB（非交互模式）...${plain}"
        case "${release}" in
        ubuntu | debian | armbian)
            apt-get update || return 1
            apt-get install -y gnupg curl ca-certificates || return 1
            curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc | gpg -o /usr/share/keyrings/mongodb-server-8.0.gpg --dearmor || return 1
            source /etc/os-release
            MONGO_DISTRO="ubuntu"
            if grep -qi '^ID=debian' /etc/os-release && ! grep -qi 'ID_LIKE=.*ubuntu' /etc/os-release; then
                MONGO_DISTRO="debian"
            fi
            VERSION_CODENAME=${VERSION_CODENAME:-$(grep -oP 'VERSION_CODENAME=\\K\w+' /etc/os-release 2>/dev/null || echo "bookworm")}
            case "${VERSION_CODENAME}" in
                noble|jammy|focal|bookworm|bullseye|trixie)
                    MONGO_CODENAME="${VERSION_CODENAME}"
                    ;;
                *)
                    if [[ "${MONGO_DISTRO}" == "debian" ]]; then
                        MONGO_CODENAME="bookworm"
                    else
                        MONGO_CODENAME="noble"
                    fi
                    ;;
            esac
            echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] https://repo.mongodb.org/apt/${MONGO_DISTRO} ${MONGO_CODENAME}/mongodb-org/8.2 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-8.2.list >/dev/null || return 1
            apt-get update || return 1
            apt-get install -y mongodb-org || return 1
            systemctl daemon-reload
            systemctl enable mongod || return 1
            systemctl start mongod || return 1
            ;;
        centos | rhel | almalinux | rocky | ol)
            cat >/etc/yum.repos.d/mongodb-org-8.2.repo <<'REPO'
[mongodb-org-8.2]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/$releasever/mongodb-org/8.2/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://www.mongodb.org/static/pgp/server-8.0.asc
REPO
            yum install -y mongodb-org || return 1
            systemctl daemon-reload
            systemctl enable mongod || return 1
            systemctl start mongod || return 1
            ;;
        fedora | amzn | virtuozzo)
            cat >/etc/yum.repos.d/mongodb-org-8.2.repo <<'REPO'
[mongodb-org-8.2]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/$releasever/mongodb-org/8.2/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://www.mongodb.org/static/pgp/server-8.0.asc
REPO
            dnf install -y mongodb-org || return 1
            systemctl daemon-reload
            systemctl enable mongod || return 1
            systemctl start mongod || return 1
            ;;
        arch | manjaro | parch)
            pacman -Sy --noconfirm mongodb || return 1
            systemctl enable --now mongodb 2>/dev/null || systemctl enable --now mongod || return 1
            ;;
        alpine)
            apk update || return 1
            apk add --no-cache mongodb mongodb-tools mongodb-openrc 2>/dev/null || apk add --no-cache mongodb || return 1
            rc-update add mongodb default 2>/dev/null || rc-update add mongod default || return 1
            rc-service mongodb start 2>/dev/null || rc-service mongod start 2>/dev/null || return 1
            ;;
        opensuse-tumbleweed)
            zypper refresh || return 1
            zypper install -y mongodb || return 1
            systemctl enable mongod 2>/dev/null || systemctl enable mongodb 2>/dev/null || return 1
            systemctl start mongod 2>/dev/null || systemctl start mongodb 2>/dev/null || return 1
            ;;
        *)
            echo -e "${red}当前发行版暂未集成 MongoDB 自动安装，请手动安装后重试${plain}"
            return 1
            ;;
        esac

        if systemctl is-active --quiet mongod 2>/dev/null || systemctl is-active --quiet mongodb 2>/dev/null || rc-service mongodb status >/dev/null 2>&1 || rc-service mongod status >/dev/null 2>&1; then
            return 0
        fi

        echo -e "${red}MongoDB 服务未能启动，请检查安装日志${plain}"
        return 1
    }

    bootstrap_selected_database() {
        local selected_db_type="$1"
        mkdir -p /etc/x-ui
        echo "XUI_DB_TYPE=${selected_db_type}" >/etc/x-ui/db-type.conf
        export XUI_DB_TYPE="${selected_db_type}"
        if [[ "${selected_db_type}" == "mongodb" ]]; then
            install_mongodb_runtime_noninteractive || return 1
            if [[ -f /etc/x-ui/x-ui.db ]]; then
                echo -e "${yellow}检测到历史 SQLite 数据库 /etc/x-ui/x-ui.db，已保留且不会删除。${plain}"
            fi

            local conf_mongo_user="${mongo_user}"
            local conf_mongo_pass="${mongo_pass}"
            local conf_mongo_db="${mongo_db:-xui}"
            local conf_mongo_host="${mongo_host:-localhost}"
            local conf_mongo_port="${mongo_port:-27017}"

            if [[ "${IS_FRESH_INSTALL}" == "1" ]]; then
                configure_mongodb_runtime_auth "${conf_mongo_user}" "${conf_mongo_pass}" "${conf_mongo_db}" || {
                    echo -e "${red}MongoDB 账户初始化失败，终止安装${plain}"
                    return 1
                }
            elif [[ -f /etc/x-ui/mongodb.conf ]]; then
                source /etc/x-ui/mongodb.conf
                conf_mongo_host=${MONGO_HOST:-$conf_mongo_host}
                conf_mongo_port=${MONGO_PORT:-$conf_mongo_port}
                conf_mongo_db=${MONGO_DB:-$conf_mongo_db}
                conf_mongo_user=${MONGO_USER:-$conf_mongo_user}
                conf_mongo_pass=${MONGO_PASS:-$conf_mongo_pass}
            fi

            cat >/etc/x-ui/mongodb.conf <<EOF
MONGO_HOST=${conf_mongo_host}
MONGO_PORT=${conf_mongo_port}
MONGO_DB=${conf_mongo_db}
MONGO_USER=${conf_mongo_user}
MONGO_PASS=${conf_mongo_pass}
EOF
            chmod 600 /etc/x-ui/mongodb.conf 2>/dev/null || true
        fi
    }

    ensure_db_type_conf() {

        mkdir -p /etc/x-ui
        if [[ ! -f /etc/x-ui/db-type.conf ]]; then
            cat <<'EOF' >/etc/x-ui/db-type.conf
XUI_DB_TYPE=sqlite
EOF
        fi
    }

    gen_random_string() {
        local length="$1"
        local random_string=$(LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | fold -w "$length" | head -n 1)
        echo "$random_string"
    }

    is_fresh_install() {
        if [[ -f "/etc/x-ui/x-ui.db" ]]; then
            return 1
        fi
        if [[ -f "/etc/x-ui/db-type.conf" || -f "/etc/x-ui/mongodb.conf" ]]; then
            return 1
        fi
        if [[ -x "/usr/local/x-ui/x-ui" ]]; then
            return 1
        fi
        return 0
    }

    configure_fresh_panel_credentials() {
        echo -e "${green}检测到全新安装，请设置面板初始化信息${plain}"
        while true; do
            read -p "请设置管理员用户名: " config_account
            [[ -n "${config_account}" ]] && break
            echo -e "${red}用户名不能为空，请重新输入。${plain}"
        done

        while true; do
            read -p "请设置管理员密码: " config_password
            [[ -n "${config_password}" ]] && break
            echo -e "${red}密码不能为空，请重新输入。${plain}"
        done

        while true; do
            read -p "请设置面板登录访问路径（输入 rd 为随机）: " config_webBasePath
            if [[ "${config_webBasePath}" == "rd" || "${config_webBasePath}" == "RD" ]]; then
                config_webBasePath=$(gen_random_string 15)
                break
            fi
            [[ -n "${config_webBasePath}" ]] && break
            echo -e "${red}访问路径不能为空，请重新输入。${plain}"
        done

        echo -e "${yellow}初始化账号: ${config_account}${plain}"
        echo -e "${yellow}初始化访问路径: ${config_webBasePath}${plain}"
    }

    configure_fresh_mongodb_credentials() {
        while true; do
            read -p "请设置 MongoDB 用户名 [默认: xui]: " mongo_user
            mongo_user=${mongo_user:-xui}
            [[ -n "${mongo_user}" ]] && break
        done
        while true; do
            read -p "请设置 MongoDB 密码（留空则随机生成）: " mongo_pass
            if [[ -z "${mongo_pass}" ]]; then
                mongo_pass=$(gen_random_string 24)
            fi
            [[ -n "${mongo_pass}" ]] && break
        done
        mongo_host="localhost"
        mongo_port="27017"
        mongo_db="xui"
    }

    configure_mongodb_runtime_auth() {
        local selected_user="$1"
        local selected_pass="$2"
        local selected_db="$3"
        local mongo_shell=""

        if command -v mongosh >/dev/null 2>&1; then
            mongo_shell="mongosh"
        elif command -v mongo >/dev/null 2>&1; then
            mongo_shell="mongo"
        else
            echo -e "${red}未检测到 mongosh/mongo，无法初始化 MongoDB 账户${plain}"
            return 1
        fi

        ${mongo_shell} --quiet "mongodb://localhost:27017" --eval "db = db.getSiblingDB('${selected_db}'); if (!db.getUser('${selected_user}')) { db.createUser({user:'${selected_user}', pwd:'${selected_pass}', roles:[{role:'readWrite', db:'${selected_db}'}]}); }" >/dev/null 2>&1 || return 1

        if [[ -f /etc/mongod.conf ]]; then
            if ! grep -q '^security:' /etc/mongod.conf; then
                cat >>/etc/mongod.conf <<'EOF'
security:
  authorization: enabled
EOF
            elif ! grep -q 'authorization:[[:space:]]*enabled' /etc/mongod.conf; then
                sed -i '/^security:/,/^[^[:space:]]/ s/authorization:[[:space:]]*.*/authorization: enabled/' /etc/mongod.conf
                if ! grep -q 'authorization:[[:space:]]*enabled' /etc/mongod.conf; then
                    sed -i '/^security:/a\  authorization: enabled' /etc/mongod.conf
                fi
            fi
        fi

        systemctl restart mongod 2>/dev/null || systemctl restart mongodb 2>/dev/null || true

        return 0
    }

    # This function will be called after package install
    config_after_install() {
        if [[ "${IS_FRESH_INSTALL}" == "1" ]]; then
            echo -e "${yellow}正在写入全新安装的面板初始化账号与访问路径...${plain}"
            if /usr/local/x-ui/x-ui setting -username "${config_account}" -password "${config_password}" -webBasePath "${config_webBasePath}"; then
                echo -e "${yellow}初始化账号与访问路径设置成功!${plain}"
            else
                echo -e "${red}初始化账号信息写入失败，请检查输入后重试。${plain}"
                return 1
            fi
            if [[ "${INSTALL_DB_TYPE}" == "mongodb" ]]; then
                echo -e "${yellow}MongoDB 初始化账号: ${mongo_user}${plain}"
                echo -e "${yellow}MongoDB 初始化密码: ${mongo_pass}${plain}"
            fi
            echo "" >&2
        else
            echo -e "${green}检测到为更新安装，保留原管理员账号、密码与访问路径。${plain}"
            echo -e "${green}如忘记登录信息，可执行 x-ui 并选择${red}数字 10${green} 查看。${plain}"
        fi
        sleep 1
        echo -e ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
        echo "" >&2
        /usr/local/x-ui/x-ui migrate
    }

    echo ""
    install_x-ui() {
        cd /usr/local/

        IS_FRESH_INSTALL=0
        if is_fresh_install; then
            IS_FRESH_INSTALL=1
            configure_fresh_panel_credentials
        fi

        choose_install_db_type
        if [[ "${IS_FRESH_INSTALL}" == "1" && "${INSTALL_DB_TYPE}" == "mongodb" ]]; then
            configure_fresh_mongodb_credentials
        fi
        bootstrap_selected_database "${INSTALL_DB_TYPE}" || { echo -e "${red}数据库初始化失败，终止安装${plain}"; exit 1; }

        # Download resources
        if [ $# == 0 ]; then
            last_version=$(choose_install_release_tag)
            if [[ ! -n "$last_version" ]]; then
                echo -e "${red}获取 X-Panel 版本失败，可能是 Github API 限制，请稍后再试${plain}" >&2
                exit 1
            fi
            echo "" >&2
            echo -e "-----------------------------------------------------"
            echo -e "${green}--------->>获取 X-Panel 最新版本：${yellow}${last_version}${plain}${green}，开始安装...${plain}"
            echo -e "-----------------------------------------------------"
            echo "" >&2
            sleep 2
            echo -e "${green}---------------->>>>>>>>>安装进度50%${plain}"
            sleep 3
            echo "" >&2
            echo -e "${green}---------------->>>>>>>>>>>>>>>>>>>>>安装进度100%${plain}"
            echo "" >&2
            sleep 2
            wget -N --no-check-certificate -O /usr/local/x-ui-linux-$(arch).tar.gz https://github.com/Sora39831/X-Panel/releases/download/${last_version}/x-ui-linux-$(arch).tar.gz
            if [[ $? -ne 0 ]]; then
                echo -e "${red}下载 X-Panel 失败, 请检查服务器是否可以连接至 GitHub？ ${plain}"
                exit 1
            fi
        else
            last_version=$1
            url="https://github.com/Sora39831/X-Panel/releases/download/${last_version}/x-ui-linux-$(arch).tar.gz"
            echo "" >&2
            echo -e "--------------------------------------------"
            echo -e "${green}---------------->>>>开始安装 X-Panel 免费基础版$1${plain}"
            echo -e "--------------------------------------------"
            echo "" >&2
            sleep 2
            echo -e "${green}---------------->>>>>>>>>安装进度50%${plain}"
            sleep 3
            echo "" >&2
            echo -e "${green}---------------->>>>>>>>>>>>>>>>>>>>>安装进度100%${plain}"
            echo "" >&2
            sleep 2
            wget -N --no-check-certificate -O /usr/local/x-ui-linux-$(arch).tar.gz ${url}
            if [[ $? -ne 0 ]]; then
                echo -e "${red}下载 X-Panel $1 失败, 请检查此版本是否存在 ${plain}"
                exit 1
            fi
        fi
        wget -O /usr/bin/x-ui-temp https://raw.githubusercontent.com/Sora39831/X-Panel/main/x-ui.sh

        # Stop x-ui service and remove old resources
        if [[ -e /usr/local/x-ui/ ]]; then
            systemctl stop x-ui
            rm /usr/local/x-ui/ -rf
        fi
        
        sleep 3
        echo -e "${green}------->>>>>>>>>>>检查并保存安装目录${plain}"
        echo "" >&2
        tar zxvf x-ui-linux-$(arch).tar.gz
        rm x-ui-linux-$(arch).tar.gz -f
        
        cd x-ui
        chmod +x x-ui
        chmod +x x-ui.sh

        # Check the system's architecture and rename the file accordingly
        if [[ $(arch) == "armv5" || $(arch) == "armv6" || $(arch) == "armv7" ]]; then
            mv bin/xray-linux-$(arch) bin/xray-linux-arm
            chmod +x bin/xray-linux-arm
        fi
        chmod +x x-ui bin/xray-linux-$(arch)

        # Update x-ui cli and se set permission
        mv -f /usr/bin/x-ui-temp /usr/bin/x-ui
        chmod +x /usr/bin/x-ui
        sleep 2
        echo -e "${green}------->>>>>>>>>>>保存成功${plain}"
        sleep 2
        echo "" >&2
        config_after_install

    ssh_forwarding() {
        # 获取 IPv4 和 IPv6 地址
        v4=$(curl -s4m8 http://ip.sb -k)
        v6=$(curl -s6m8 http://ip.sb -k)
        local existing_webBasePath=$(/usr/local/x-ui/x-ui setting -show true | grep -Eo 'webBasePath（访问路径）: .+' | awk '{print $2}') 
        local existing_port=$(/usr/local/x-ui/x-ui setting -show true | grep -Eo 'port（端口号）: .+' | awk '{print $2}') 
        local existing_cert=$(/usr/local/x-ui/x-ui setting -getCert true | grep -Eo 'cert: .+' | awk '{print $2}')
        local existing_key=$(/usr/local/x-ui/x-ui setting -getCert true | grep -Eo 'key: .+' | awk '{print $2}')

        if [[ -n "$existing_cert" && -n "$existing_key" ]]; then
            echo -e "${green}面板已安装证书采用SSL保护${plain}"
            echo "" >&2
            local existing_cert=$(/usr/local/x-ui/x-ui setting -getCert true | grep -Eo 'cert: .+' | awk '{print $2}')
            domain=$(basename "$(dirname "$existing_cert")")
            echo -e "${green}登录访问面板URL: https://${domain}:${existing_port}${green}${existing_webBasePath}${plain}"
        fi
        echo "" >&2
        if [[ -z "$existing_cert" && -z "$existing_key" ]]; then
            echo -e "${red}警告：未找到证书和密钥，面板不安全！${plain}"
            echo "" >&2
            echo -e "${green}------->>>>请按照下述方法设置〔ssh转发〕<<<<-------${plain}"
            echo "" >&2

            # 检查 IP 并输出相应的 SSH 和浏览器访问信息
            if [[ -z $v4 ]]; then
                echo -e "${green}1、本地电脑客户端转发命令：${plain} ${blue}ssh  -L [::]:15208:127.0.0.1:${existing_port}${blue} root@[$v6]${plain}"
                echo "" >&2
                echo -e "${green}2、请通过快捷键【Win + R】调出运行窗口，在里面输入【cmd】打开本地终端服务${plain}"
                echo "" >&2
                echo -e "${green}3、请在终端中成功输入服务器的〔root密码〕，注意区分大小写，用以上命令进行转发${plain}"
                echo "" >&2
                echo -e "${green}4、请在浏览器地址栏复制${plain} ${blue}[::1]:15208${existing_webBasePath}${plain} ${green}进入〔X-Panel面板〕登录界面"
                echo "" >&2
                echo -e "${red}注意：若不使用〔ssh转发〕请为X-Panel面板配置安装证书再行登录管理后台${plain}"
            elif [[ -n $v4 && -n $v6 ]]; then
                echo -e "${green}1、本地电脑客户端转发命令：${plain} ${blue}ssh -L 15208:127.0.0.1:${existing_port}${blue} root@$v4${plain} ${yellow}或者 ${blue}ssh  -L [::]:15208:127.0.0.1:${existing_port}${blue} root@[$v6]${plain}"
                echo "" >&2
                echo -e "${green}2、请通过快捷键【Win + R】调出运行窗口，在里面输入【cmd】打开本地终端服务${plain}"
                echo "" >&2
                echo -e "${green}3、请在终端中成功输入服务器的〔root密码〕，注意区分大小写，用以上命令进行转发${plain}"
                echo "" >&2
                echo -e "${green}4、请在浏览器地址栏复制${plain} ${blue}127.0.0.1:15208${existing_webBasePath}${plain} ${yellow}或者${plain} ${blue}[::1]:15208${existing_webBasePath}${plain} ${green}进入〔X-Panel面板〕登录界面"
                echo "" >&2
                echo -e "${red}注意：若不使用〔ssh转发〕请为X-Panel面板配置安装证书再行登录管理后台${plain}"
            else
                echo -e "${green}1、本地电脑客户端转发命令：${plain} ${blue}ssh -L 15208:127.0.0.1:${existing_port}${blue} root@$v4${plain}"
                echo "" >&2
                echo -e "${green}2、请通过快捷键【Win + R】调出运行窗口，在里面输入【cmd】打开本地终端服务${plain}"
                echo "" >&2
                echo -e "${green}3、请在终端中成功输入服务器的〔root密码〕，注意区分大小写，用以上命令进行转发${plain}"
                echo "" >&2
                echo -e "${green}4、请在浏览器地址栏复制${plain} ${blue}127.0.0.1:15208${existing_webBasePath}${plain} ${green}进入〔X-Panel面板〕登录界面"
                echo "" >&2
                echo -e "${red}注意：若不使用〔ssh转发〕请为X-Panel面板配置安装证书再行登录管理后台${plain}"
                echo "" >&2
            fi
        fi
    }
    # 执行ssh端口转发
    ssh_forwarding

        cp -f x-ui.service /etc/systemd/system/
        systemctl daemon-reload
        systemctl enable x-ui
        ensure_db_type_conf
        systemctl start x-ui
        systemctl stop warp-go >/dev/null 2>&1
        wg-quick down wgcf >/dev/null 2>&1
        ipv4=$(curl -s4m8 ip.p3terx.com -k | sed -n 1p)
        ipv6=$(curl -s6m8 ip.p3terx.com -k | sed -n 1p)
        systemctl start warp-go >/dev/null 2>&1
        wg-quick up wgcf >/dev/null 2A>&1

        echo "" >&2
        echo -e "------->>>>${green}X-Panel 免费基础版 ${last_version}${plain}<<<<安装成功，正在启动..."
        sleep 1
        echo "" >&2
        echo -e "         ---------------------"
        echo -e "         |${green}X-Panel 控制菜单用法 ${plain}|${plain}"
        echo -e "         |  ${yellow}一个更好的面板   ${plain}|${plain}"   
        echo -e "         | ${yellow}基于Xray Core构建 ${plain}|${plain}"  
        echo -e "--------------------------------------------"
        echo -e "x-ui              - 进入管理脚本"
        echo -e "x-ui start        - 启动 X-Panel 面板"
        echo -e "x-ui stop         - 关闭 X-Panel 面板"
        echo -e "x-ui restart      - 重启 X-Panel 面板"
        echo -e "x-ui status       - 查看 X-Panel 状态"
        echo -e "x-ui settings     - 查看当前设置信息"
        echo -e "x-ui enable       - 启用 X-Panel 开机启动"
        echo -e "x-ui disable      - 禁用 X-Panel 开机启动"
        echo -e "x-ui log          - 查看 X-Panel 运行日志"
        echo -e "x-ui banlog       - 检查 Fail2ban 禁止日志"
        echo -e "x-ui update       - 更新 X-Panel 面板"
        echo -e "x-ui custom       - 自定义 X-Panel 版本"
        echo -e "x-ui install      - 安装 X-Panel 面板"
        echo -e "x-ui uninstall    - 卸载 X-Panel 面板"
        echo -e "--------------------------------------------"
        echo "" >&2
        # if [[ -n $ipv4 ]]; then
        #    echo -e "${yellow}面板 IPv4 访问地址为：${green}http://$ipv4:${config_port}/${config_webBasePath}${plain}"
        # fi
        # if [[ -n $ipv6 ]]; then
        #    echo -e "${yellow}面板 IPv6 访问地址为：${green}http://[$ipv6]:${config_port}/${config_webBasePath}${plain}"
        # fi
        #    echo -e "请自行确保此端口没有被其他程序占用，${yellow}并且确保${red} ${config_port} ${yellow}端口已放行${plain}"
        sleep 3
        echo -e ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
        echo "" >&2
        echo -e "${yellow}----->>>X-Panel面板和Xray启动成功<<<-----${plain}"
    }

    # 设置VPS中的时区/时间为【上海时间】
    sudo timedatectl set-timezone Asia/Shanghai

    install_base
    install_x-ui $1
    echo ""
    echo -e "----------------------------------------------"
    sleep 4
    info=$(/usr/local/x-ui/x-ui setting -show true)
    echo -e "${info}${plain}"
    echo ""
    echo -e "若您忘记了上述面板信息，后期可通过x-ui命令进入脚本${red}输入数字〔10〕选项获取${plain}"
    echo ""
    echo -e "----------------------------------------------"
    echo ""
    sleep 2
    echo -e "${green}安装/更新完成，若在使用过程中有任何问题${plain}"
    echo -e "${yellow}请先描述清楚所遇问题加〔X-Panel面板〕交流群${plain}"
    echo -e "${yellow}在TG群中${red} https://t.me/XUI_CN ${yellow}截图进行反馈${plain}"
    echo ""
    echo -e "----------------------------------------------"
    echo ""
    echo -e "${green}〔X-Panel面板〕项目地址：${yellow}https://github.com/Sora39831/X-Panel${plain}" 
    echo ""
    echo -e "${green} 详细安装教程：${yellow}https://xeefei.blogspot.com/2025/09/x-panel.html${plain}"
    echo ""
    echo -e "----------------------------------------------"
    echo ""
    echo -e "-------------->>>>>>>赞 助 推 广 区<<<<<<<<-------------------"
    echo ""
    echo -e "${green}1、搬瓦工GIA高端线路：${yellow}https://bandwagonhost.com/aff.php?aff=75015${plain}"
    echo ""
    echo -e "${green}2、Dmit高端GIA线路：${yellow}https://www.dmit.io/aff.php?aff=9326${plain}"
    echo ""
    echo -e "${green}3、Gomami亚太顶尖优化线路：${yellow}https://gomami.io/aff.php?aff=174${plain}"
    echo ""
    echo -e "${green}4、ISIF优质亚太优化线路：${yellow}https://cloud.isif.net/login?affiliation_code=333${plain}"
    echo ""
    echo -e "${green}5、ZoroCloud全球优质原生家宽&住宅双lSP，跨境首选：${yellow}https://my.zorocloud.com/aff.php?aff=1072${plain}"
    echo ""
    echo -e "${green}6、三网直连 IEPL / IPLC 直播流量转发：${yellow}https://idc333.top/#register/BCUZXNELNO${plain}"
    echo ""
    echo -e "${green}7、Bagevm优质落地鸡（原生IP全解锁）：${yellow}https://www.bagevm.com/aff.php?aff=754${plain}"
    echo ""
    echo -e "${green}8、白丝云〔4837线路〕实惠量大管饱：${yellow}https://cloudsilk.io/aff.php?aff=706${plain}"
    echo ""
    echo -e "${green}9、RackNerd极致性价比机器：${yellow}https://my.racknerd.com/aff.php?aff=15268&pid=912${plain}"
    echo ""
    echo -e "----------------------------------------------"
    echo ""
}

# 免费版安装逻辑函数 (install_free_version) 结束

# ----------------------------------------------------------
# 脚本主菜单
# ----------------------------------------------------------
main_menu() {
    echo -e "${green}======================================================${plain}"
    echo -e " 欢迎使用 ${yellow}〔X-Panel 面板〕${plain} 一键安装脚本"
    echo -e "${green}======================================================${plain}"
    echo ""
    echo -e "请选择您要安装的版本:"
    echo ""
    echo -e "  ${green}1)${plain} 安装 ${yellow}〔X-Panel 面板〕免费基础版${plain} (GitHub 开源项目)"
    echo ""
    echo -e "  ${green}2)${plain} 安装 ${yellow}〔X-Panel 面板〕付费Pro版${plain} (需要购买授权码)"
    echo ""
    read -p "请输入您的选择 (1 或 2): " version_choice
    echo ""
    
    case "$version_choice" in
        1)
            # 如果选择1，调用免费版函数
            install_free_version
            ;;
        2)
            # 如果选择2，调用付费版函数
            install_paid_version
            ;;
        *)
            echo -e "${red}输入无效, 退出安装。${plain}"
            exit 1
            ;;
    esac
}

# ----------------------------------------------------------
# 脚本执行入口
# ----------------------------------------------------------
clear
main_menu
