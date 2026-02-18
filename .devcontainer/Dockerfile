FROM fedora:45

RUN dnf install -y \
    git \
    gcc \
    golang \
    make \
    openssh-clients \
    curl \
    jq \
    nodejs \
    npm \
    && dnf clean all

RUN npm install -g @anthropic-ai/claude-code
