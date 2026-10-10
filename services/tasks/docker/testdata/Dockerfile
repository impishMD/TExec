ARG TEXEC_TEST_BASE=texec:docker-preview
FROM ${TEXEC_TEST_BASE}
COPY --chown=1001:0 . /opt/texec-test-repo
RUN printf '%s\n' 'fixture-vault-password' > /tmp/fixture-vault-password && \
    ansible-vault encrypt --vault-password-file /tmp/fixture-vault-password /opt/texec-test-repo/vault.yml && \
    rm /tmp/fixture-vault-password && \
    git -C /opt/texec-test-repo init -b main && \
    git -C /opt/texec-test-repo add . && \
    git -C /opt/texec-test-repo -c user.name=Test -c user.email=test@example.invalid commit -m fixture
