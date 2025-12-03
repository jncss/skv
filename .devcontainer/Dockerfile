# ==============================================================================
# Dockerfile: Entorn de Desenvolupament Complet
# Ubuntu 24.04 + Go (goenv) + Node.js (nvm) + Yarn + Oh My Zsh + PowerLevel10k
# ==============================================================================

# 1. Imatge Base
# -----------------
# Utilitzem la versió LTS (Long-Term Support) més recent d'Ubuntu per a més estabilitat.
FROM ubuntu:24.04

# ==============================================================================
# 2. Arguments i Variables d'Entorn
# -----------------------------------
# Defineix les versions com a arguments per poder modificar-les fàcilment.
ARG GO_VERSION=1.25.1
ARG NODE_VERSION=22      # Pot ser una versió major (22) o específica (22.4.0)
ARG NVM_VERSION=0.39.7

# Evita diàlegs interactius durant la instal·lació de paquets.
ENV DEBIAN_FRONTEND=noninteractive

# ==============================================================================
# 3. Instal·lació de Dependències del Sistema
# ---------------------------------------------
# Actualitza els paquets i instal·la les eines bàsiques necessàries.
RUN apt-get update && \
    apt-get upgrade -y && \
    # Habilita l'arquitectura i386 per wine32 i actualitza només una vegada més
    dpkg --add-architecture i386 && \
    apt-get update && \
    apt-get install -y \
        # Editors i eines bàsiques
        vim \
        nano \
        # Wine per Windows
        wine \
        wine32:i386 \
        winetricks \
        # Shell i eines de xarxa
        zsh \
        curl \
        wget \
        git \
        # Compiladors i build tools
        build-essential \
        # Certificats i locales
        ca-certificates \
        locales \
        # Utilitats de xarxa i sistema
        iputils-ping \
        sudo \
        # Fonts
        fontconfig \
        # Ruby per colorls
        ruby-full && \
    # Configura UTF-8 locale per suport Unicode
    sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && \
    locale-gen && \
    # Neteja la memòria cau d'APT per reduir la mida de la imatge.
    rm -rf /var/lib/apt/lists/*

# Esborra l'usuari ubuntu si existeix
RUN id ubuntu &>/dev/null && userdel -r ubuntu || true

# ==============================================================================
# 3.1. Instal·lació d'Inno Setup (versió preinstal·lada)
# --------------------------------------------------------
# Copia Inno Setup preinstal·lat des del .devcontainer
COPY Inno_Setup_6 /opt/innosetup
RUN chmod +x /opt/innosetup/*.exe

# ==============================================================================
# 5. Instal·lació de la font MesloLGS NF
# ----------------------------------------
RUN mkdir -p /usr/share/fonts/truetype/meslo && \
    # Regular
    curl -fsSL -o /usr/share/fonts/truetype/meslo/MesloLGS-NF-Regular.ttf \
        https://github.com/ryanoasis/nerd-fonts/raw/master/patched-fonts/Meslo/S/Regular/MesloLGSNerdFont-Regular.ttf && \
    # Bold
    curl -fsSL -o /usr/share/fonts/truetype/meslo/MesloLGS-NF-Bold.ttf \
        https://github.com/ryanoasis/nerd-fonts/raw/master/patched-fonts/Meslo/S/Bold/MesloLGSNerdFont-Bold.ttf && \
    # Italic
    curl -fsSL -o /usr/share/fonts/truetype/meslo/MesloLGS-NF-Italic.ttf \
        https://github.com/ryanoasis/nerd-fonts/raw/master/patched-fonts/Meslo/S/Italic/MesloLGSNerdFont-Italic.ttf && \
    # Bold Italic
    curl -fsSL -o /usr/share/fonts/truetype/meslo/MesloLGS-NF-BoldItalic.ttf \
        https://github.com/ryanoasis/nerd-fonts/raw/master/patched-fonts/Meslo/S/Bold-Italic/MesloLGSNerdFont-BoldItalic.ttf && \
    fc-cache -fv

# ==============================================================================
# 6. Instal·lació d'Oh My Zsh i PowerLevel10k
# ---------------------------------------------
RUN RUNZSH=no CHSH=no KEEP_ZSHRC=yes bash -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" && \
    git clone --depth=1 https://github.com/romkatv/powerlevel10k.git /root/.oh-my-zsh/custom/themes/powerlevel10k

# ==============================================================================
# 7. Instal·lació de goenv
# --------------------------
RUN git clone https://github.com/syndbg/goenv.git /root/.goenv

# ==============================================================================
# 8. Creació i configuració de l'usuari dev
# -------------------------------------------
RUN useradd -m -s /usr/bin/zsh dev && \
    usermod -aG sudo dev && \
    echo 'dev ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/dev && \
    chmod 0440 /etc/sudoers.d/dev && \
    # Copia configuracions de root a dev
    cp -r /root/.oh-my-zsh /home/dev/.oh-my-zsh && \
    cp -r /root/.goenv /home/dev/.goenv && \
    chown -R dev:dev /home/dev

# Copia fitxers de configuració personalitzats
COPY .zshrc /home/dev/.zshrc
COPY .p10k.zsh /home/dev/.p10k.zsh
RUN chown dev:dev /home/dev/.zshrc /home/dev/.p10k.zsh

# Configuració SSH - creació del directori .ssh i còpia de claus RSA
RUN mkdir -p /home/dev/.ssh && \
    chmod 700 /home/dev/.ssh && \
    chown dev:dev /home/dev/.ssh
COPY id_rsa /home/dev/.ssh/id_rsa
COPY id_rsa.pub /home/dev/.ssh/id_rsa.pub
RUN chown dev:dev /home/dev/.ssh/id_rsa /home/dev/.ssh/id_rsa.pub && \
    chmod 600 /home/dev/.ssh/id_rsa && \
    chmod 644 /home/dev/.ssh/id_rsa.pub

# Crea script ISCC per executar Inno Setup Compiler
COPY <<'EOF' /home/dev/.local/bin/iscc
#!/bin/bash
# Script per executar Inno Setup Compiler amb la versió preinstal·lada

if [ $# -eq 0 ]; then
    echo "Ús: iscc <fitxer.iss> [opcions]"
    echo "Exemple: iscc myproject.iss"
    exit 1
fi

# Ubicació d'Inno Setup preinstal·lat
ISCC_EXE="/opt/innosetup/ISCC.exe"

if [ ! -f "$ISCC_EXE" ]; then
    echo "ERROR: No es troba ISCC.exe a $ISCC_EXE"
    echo "Verifica que Inno Setup s'hagi copiat correctament."
    exit 1
fi

# Configura Wine amb debugging mínim
export WINEDEBUG=-all
export WINEPREFIX="$HOME/.wine"

# Inicialitza Wine si no està configurat
if [ ! -d "$HOME/.wine" ]; then
    echo "Inicialitzant Wine..."
    wineboot --init >/dev/null 2>&1
    sleep 3
fi

# Executa ISCC.exe
echo "Compilant amb Inno Setup..."
wine "$ISCC_EXE" "$@"
EOF

# Configura permisos dels scripts (després de crear-los)
RUN chmod +x /home/dev/.local/bin/iscc && \
    chown dev:dev /home/dev/.local/bin/iscc && \
    echo "Inno Setup script configured. ISCC.exe available at /opt/innosetup/"

# ==============================================================================
# 9.1. Instal·lació de colorls (com a root)
# ----------------------------
RUN gem install colorls

# Canvi a l'usuari dev per a la resta d'operacions
USER dev
WORKDIR /home/dev

# Defineix variables d'entorn per a l'usuari dev
ENV NVM_DIR="/home/dev/.nvm"
ENV GOENV_ROOT="/home/dev/.goenv"

# Configuració global de Git per l'usuari dev
RUN git config --global user.email "dev@systembcn.com" && \
    git config --global user.name "Dev User" && \
    git config --global init.defaultBranch main && \
    git config --global pull.rebase false

# ==============================================================================
# 9. Configuració d'entorn i variables
# -------------------------------------
# ==============================================================================
# 9. Configuració d'entorn i variables
# -------------------------------------
RUN echo '\n# Variables d'\''entorn per Go' >> ~/.bashrc && \
    echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bashrc && \
    echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bashrc && \
    echo 'eval "$(goenv init -)"' >> ~/.bashrc && \
    \
    echo '\n# Variables d'\''entorn per Node.js' >> ~/.bashrc && \
    echo 'export NVM_DIR="$HOME/.nvm"' >> ~/.bashrc && \
    echo '[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm' >> ~/.bashrc && \
    \
    echo '\n# Variables d'\''entorn per colors i fonts' >> ~/.bashrc && \
    echo 'export TERM=xterm-256color' >> ~/.bashrc && \
    echo 'export COLORTERM=truecolor' >> ~/.bashrc && \
    echo 'export LC_ALL=en_US.UTF-8' >> ~/.bashrc && \
    echo 'export LANG=en_US.UTF-8' >> ~/.bashrc && \
    echo 'export PATH="$HOME/.local/share/gem/ruby/3.2.0/bin:$HOME/.local/bin:$PATH"' >> ~/.bashrc

RUN echo '\n# Variables d'\''entorn per Go' >> ~/.zshrc && \
    echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.zshrc && \
    echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.zshrc && \
    echo 'eval "$(goenv init -)"' >> ~/.zshrc && \
    \
    echo '\n# Variables d'\''entorn per Node.js' >> ~/.zshrc && \
    echo 'export NVM_DIR="$HOME/.nvm"' >> ~/.zshrc && \
    echo '[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm' >> ~/.zshrc && \
    \
    echo '\n# Variables d'\''entorn per colors i fonts' >> ~/.zshrc && \
    echo 'export TERM=xterm-256color' >> ~/.zshrc && \
    echo 'export COLORTERM=truecolor' >> ~/.zshrc && \
    echo 'export LC_ALL=en_US.UTF-8' >> ~/.zshrc && \
    echo 'export LANG=en_US.UTF-8' >> ~/.zshrc && \
    echo 'export POWERLEVEL9K_MODE="nerdfont-complete"' >> ~/.zshrc && \
    echo 'export ZSH_FONT="MesloLGS NF"' >> ~/.zshrc && \
    echo 'export PATH="$HOME/.local/share/gem/ruby/3.2.0/bin:$HOME/.local/bin:$PATH"' >> ~/.zshrc

# ==============================================================================
# 11. Instal·lació de NVM i Node.js
# -----------------------------------
RUN curl -o- "https://raw.githubusercontent.com/nvm-sh/nvm/v${NVM_VERSION}/install.sh" | bash && \
    bash -c '. $HOME/.nvm/nvm.sh && nvm install ${NODE_VERSION} && nvm alias default ${NODE_VERSION} && corepack enable'

# ==============================================================================
# 11.1. Instal·lació de Vue CLI
# ------------------------------
RUN bash -c '. $HOME/.nvm/nvm.sh && npm install -g @vue/cli'

# ==============================================================================
# 12. Instal·lació de Go amb goenv
# ----------------------------------
RUN bash -c 'export GOENV_ROOT="$HOME/.goenv" && export PATH="$GOENV_ROOT/bin:$PATH" && eval "$(goenv init -)" && goenv install ${GO_VERSION} && goenv global ${GO_VERSION}'

# Configurar repositoris privats
RUN bash -c 'export GOENV_ROOT="$HOME/.goenv" && export PATH="$GOENV_ROOT/bin:$PATH" && eval "$(goenv init -)" && go env -w GOPRIVATE="git.systembcn.com" GOINSECURE="git.systembcn.com"'
    
# ==============================================================================
# 13. Verificació de les versions
# --------------------------------
RUN . ~/.nvm/nvm.sh && \
    export GOENV_ROOT="$HOME/.goenv" && \
    export PATH="$GOENV_ROOT/bin:$HOME/.local/share/gem/ruby/3.2.0/bin:$PATH" && \
    eval "$(goenv init -)" && \
    \
    echo '--- Versions instal·lades ---' && \
    go version && \
    node -v && \
    yarn -v && \
    vue --version && \
    colorls --version && \
    echo '--------------------------'

# ==============================================================================
# 14. Configuració Final
# -----------------------
WORKDIR /app
CMD [ "zsh" ]


