⏺ Plano: Persistência em arquivo para o fake-platform-api (sem banco de dados)

Objetivo

Permitir que a Fake API sobreviva a restarts do container/processo (atualizações de imagem, systemctl restart, queda da instância EC2) mantendo agents,
commands, deployments, desired states e counters, sem introduzir dependência de banco de dados — usando persistência em arquivo JSON local (volume
Docker).

Isso resolveria, de quebra, o problema operacional que vimos (agent not found após restart, perda de desired_state versão).
  
---
Decisão de design

- Formato: snapshot único em JSON, espelhando o estado interno do Store (Agents, Commands, Deployments, DesiredStates, Reports, Counters).
- Quando salvar: snapshot periódico (ex.: a cada 2s, configurável) rodando em goroutine + flush no shutdown gracioso (SIGTERM/SIGINT). Evita instrumentar
  cada operação de mutação individualmente.
- Quando carregar: na inicialização (store.New), se o arquivo existir e DEVEX_FAKE_STATE_FILE estiver configurado, restaura o snapshot; caso contrário,
  inicia vazio (comportamento atual preservado).
- Escrita atômica: grava em arquivo temporário (*.tmp) e usa os.Rename para evitar corrupção em caso de crash durante a escrita.
- Opt-in: se DEVEX_FAKE_STATE_FILE não for definido, a API continua 100% em memória (sem mudança de comportamento — preserva a regra "Store em memória"
  do MVP para quem não precisa de persistência).

  ---
Milestones

M1 — Config e contrato de ativação
- Novo env var DEVEX_FAKE_STATE_FILE (ex.: /data/fake-platform-api/state.json); vazio = desabilitado.
- Novo env var opcional DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS (default 2).
- Atualizar internal/config/config.go e docs/specs/09-docker-compose-dev.md.

M2 — Snapshot/Restore no Store
- Novo arquivo internal/store/snapshot.go:
    - type Snapshot struct{...} (espelha campos exportáveis do Store: Agents, Commands, Deployments, DesiredStates, DesiredStateReports, Counters, etc.)
    - func (s *Store) Snapshot() Snapshot — copia sob RLock.
    - func (s *Store) Restore(snap Snapshot) — popula sob Lock, chamado só na inicialização (sem concorrência).

M3 — Camada de persistência em arquivo
- Novo pacote internal/persistence/:
    - func Load(path string) (*store.Snapshot, error) — lê e faz json.Unmarshal; se arquivo não existe, retorna nil, nil.
    - func Save(path string, snap store.Snapshot) error — serializa, grava em path+".tmp", depois os.Rename.
- Sem dependências externas — apenas encoding/json e os.

M4 — Integração no ciclo de vida (main.go)
- Na inicialização: se cfg.StateFile != "", tenta persistence.Load; se houver snapshot, store.Restore.
- Inicia goroutine de snapshot periódico (time.Ticker) que chama st.Snapshot() + persistence.Save().
- Registra handler de SIGTERM/SIGINT (signal.NotifyContext) para fazer um save final antes de encerrar — importante porque systemctl stop/docker stop
  envia SIGTERM.

M5 — Ajuste do /testing/reset
- Regra 14 do CLAUDE.md: reset limpa tudo, inclusive counters. Deve também:
    - Zerar o snapshot em memória (já faz isso).
    - Forçar um save imediato (ou apagar o arquivo), para que um restart subsequente não restaure o estado anterior ao reset.

M6 — Docker / Compose
- Adicionar volume no docker-compose.dev.yml e no Dockerfile/exemplo de docker run:
  volumes:
    - ./tmp/fake-platform-api-data:/data
      environment:
      DEVEX_FAKE_STATE_FILE: "/data/state.json"
- Atualizar docs/specs/09-docker-compose-dev.md e o exemplo de docker run que já passamos para a EC2.

M7 — Testes
- internal/store/snapshot_test.go: round-trip Snapshot → Restore preserva igualdade profunda do estado.
- internal/persistence/persistence_test.go: Save+Load preserva dados; Load em arquivo inexistente retorna nil, nil; escrita atômica não deixa arquivo
  .tmp órfão.
- Teste de integração leve em cmd/ ou internal/api: cria estado, simula "restart" (novo Store carregando do mesmo arquivo), confirma que
  agents/deployments/desired-state continuam acessíveis.

M8 — Documentação
- Atualizar docs/specs/04-state-model.md (mencionar persistência opcional em arquivo) e docs/specs/99-decisions-and-clarifications.md com a decisão
  (persistência em arquivo, opt-in, sem DB).
- Atualizar README.md/CLAUDE.md se necessário.

  ---
Riscos e cuidados

- Concorrência: snapshot deve copiar sob lock e serializar fora do lock, para não travar requests durante I/O em disco.
- Compatibilidade de schema: se a estrutura interna mudar entre versões da imagem, um snapshot antigo pode falhar ao restaurar — sugiro incluir um campo
  schema_version no Snapshot e, em caso de mismatch, logar um warning e iniciar vazio (fail-safe, não fail-fatal).
- IDs/Counters: é essencial restaurar Counters corretamente para não colidir/reaproveitar IDs já usados (ex.: agent-dev-gateway-001 reaparecendo com
  dados divergentes).

  ---
Quer que eu comece a implementar pelo M1+M2 (config + snapshot/restore no store), ou prefere ajustar algo no plano antes?