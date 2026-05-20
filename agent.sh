#!/usr/bin/env bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
cyan='\033[0;36m'
plain='\033[0m'

#Add some basic function here
function LOGD() {
    echo -e "${yellow}[DEG] $* ${plain}"
}

function LOGE() {
    echo -e "${red}[ERR] $* ${plain}"
}

function LOGI() {
    echo -e "${green}[INF] $* ${plain}"
}

agent_exe_filename="${AGENT_EXE_FILENAME:=agent.exe}"
main_filepath="${AGENT_MAIN_FOLDER:=./cmd/claude-analogue/main.go}"

# 0: exists, 1: doesn't exist,
check_exe() {
    LOGI "checking exe"
    [[ -f "$agent_exe_filename" ]]
}

build() {
    LOGI "executable not found: start building"
    go build -o "${agent_exe_filename}" "${main_filepath}"
}

agent_coproc_pid() {
    echo "${AGENT_PROC_PID:-${COPROC_PID:-}}"
}

agent_is_running() {
    local p
    p="$(agent_coproc_pid)"
    [[ -n "$p" ]] && kill -0 "$p" 2>/dev/null
}

close_agent_fds() {
    if [[ -n "${AGENT_PROC[0]:-}" ]]; then
        eval "exec ${AGENT_PROC[0]}<&-" 2>/dev/null || true
    fi
    if [[ -n "${AGENT_PROC[1]:-}" ]]; then
        eval "exec ${AGENT_PROC[1]}>&-" 2>/dev/null || true
    fi
    unset AGENT_PROC
    unset AGENT_PROC_PID
}

drain_agent_banner() {
    local _
    if IFS= read -r -t 8 -u "${AGENT_PROC[0]}" _ 2>/dev/null; then
        :
    fi
}

start() {
    if agent_is_running; then
        LOGI "агент уже запущен (pid $(agent_coproc_pid))"
        return 0
    fi
    close_agent_fds

    check_exe
    if [[ $? == 1 ]]; then
        build
    fi

    LOGI "запуск агента…"
    coproc AGENT_PROC { ./"$agent_exe_filename"; }
    sleep 1
    drain_agent_banner
    if agent_is_running; then
        LOGI "агент запущен (pid $(agent_coproc_pid))"
    else
        LOGE "не удалось подтвердить, что агент жив (проверьте сборку и .env)"
    fi
}

stop() {
    local p
    p="$(agent_coproc_pid)"
    if ! agent_is_running; then
        LOGI "агент не запущен"
        close_agent_fds
        return 0
    fi
    LOGI "остановка агента (pid ${p})…"
    kill "${p}" 2>/dev/null || true
    wait "${p}" 2>/dev/null || true
    close_agent_fds
    LOGI "агент остановлен"
}

read_agent_stdout_answer() {
    local idle_sec="${1:-5}"
    local out="" line=""
    if ! IFS= read -r -t 600 -u "${AGENT_PROC[0]}" line; then
        LOGE "таймаут или EOF при ожидании ответа агента"
        return 1
    fi
    out=$line
    while IFS= read -r -t "${idle_sec}" -u "${AGENT_PROC[0]}" line; do
        out+=$'\n'"$line"
    done
    printf '%s\n' "$out"
}

prompt_interactive() {
    if ! agent_is_running; then
        LOGI "агент не запущен — выполняю start"
        start
        if ! agent_is_running; then
            LOGE "не удалось запустить агента"
            return 1
        fi
    fi

    echo -ne "${cyan}Запрос агенту: ${plain}"
    local user_prompt
    IFS= read -r user_prompt || return 1
    if [[ -z "${user_prompt// }" ]]; then
        LOGE "пустой запрос"
        return 1
    fi

    echo "$user_prompt" >&"${AGENT_PROC[1]}"
    read_agent_stdout_answer 5
}

menu() {
    while true; do
        echo ""
        echo -e "${cyan}========== Меню агента ==========${plain}"
        echo "  1 | start  — запустить агента (coproc) и вернуться в меню"
        echo "  2 | stop   — остановить агента"
        echo "  3 | prompt — ввести запрос и дождаться ответа"
        echo "  0 | quit   — выход (агент будет остановлен)"
        echo -ne "${cyan}Выбор: ${plain}"
        local choice
        IFS= read -r choice || break
        choice="${choice,,}"
        choice="${choice//[[:space:]]/}"

        case "$choice" in
            1 | start)
                start
                ;;
            2 | stop)
                stop
                ;;
            3 | prompt)
                prompt_interactive
                ;;
            0 | quit | exit | q)
                stop
                LOGI "выход"
                break
                ;;
            *)
                LOGE "неизвестная команда: ${choice}"
                ;;
        esac
    done
}

menu
