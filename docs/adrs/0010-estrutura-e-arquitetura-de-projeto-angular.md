# 0010 - Padronização de Arquitetura de Pastas e Layout do Projeto Angular Frontend

## Status

Aceito.

## Data

2026-10-28

Atualizado em: 2026-10-29

## Contexto e Problema

Para a interface SPA da aplicação Task Manager, é necessário estabelecer uma estrutura de arquivos e diretórios que
garanta escalabilidade, alta manutenibilidade e separação clara de responsabilidades. Projetos Angular sem uma
arquitetura orientada a domínios costumam concentrar componentes em pastas genéricas (`components/`, `services/`),
misturando lógicas globais (como autenticação e interceptores) com lógicas específicas de negócio, além de acumular
acoplamento entre telas.

## Fatores de Decisão

* **Angular Moderno (Standalone Architecture):** Adoção das melhores práticas do Angular recente, priorizando Standalone
  Components, Signals para reatividade local e eliminação do boilerplate de `NgModules`.
* **Organização Orientada a Funcionalidades (*Feature-First*):** Agrupar componentes, serviços, modelos e rotas por
  domínio de negócio (ex: `auth`, `tasks`, `notifications`).
* **Isolamento de Efeitos Globais (*Core Layer*):** Centralizar interceptores HTTP (`x-correlation-id`, JWT), *guards*
  de rota e serviços singleton.
* **Reaproveitamento de UI (*Shared Layer*):** Isolar componentes visuais genéricos (botões, modais, *cards*, *pipes*,
  diretivas) sem dependências com lógicas de negócio específicas.
* **Padrão Smart vs. Dumb Components:** Separar componentes de página/contêiner (*Smart Components* - gerenciam estado e
  chamadas de API) de componentes puramente visuais (*Dumb/Presentational Components* - recebem `@Input` e emitem
  `@Output`).

## Opções Consideradas

* **Opção 1:** Arquitetura Plana/Por Tipo (`components/`, `services/`, `models/` na raiz de `src/app`).
* **Opção 2:** Arquitetura Tradicional baseada em `NgModules` por funcionalidade (`AuthModule`, `TasksModule`).
* **Opção 3:** Arquitetura Modular *Feature-First* baseada em Standalone Components (`core/`, `shared/`, `features/`).

## Resultado da Decisão

Opção escolhida: "**Opção 3: Arquitetura Modular Feature-First baseada em Standalone Components**", pois alinha o projeto
às recomendações oficiais modernas do Angular, simplifica o carregamento sob demanda (*lazy loading*) e garante
isolamento entre telas.

### Estrutura de Pastas Padronizada para o Frontend Angular

```text
frontend/
├── src/
│   ├── app/
│   │   ├── core/                       # Lógicas singleton globais da aplicação
│   │   │   ├── guards/                 # Guards de navegação (auth.guard.ts)
│   │   │   ├── http/                   # Utilitários HTTP (http-error.utils.ts)
│   │   │   ├── interceptors/           # Interceptores HTTP (observability.interceptor.ts, jwt.interceptor.ts)
│   │   │   ├── telemetry/              # Provider e exportador OpenTelemetry do browser
│   │   │   ├── models/                 # Interfaces/Types do ecossistema global
│   │   │   └── services/               # Serviços globais (auth.service.ts)
│   │   │
│   │   ├── shared/                     # UI genérica sem regra de negócio
│   │   │   ├── components/             # button, modal, navbar, loading-spinner
│   │   │   ├── directives/             # Diretivas genéricas de DOM
│   │   │   └── pipes/                  # Pipes customizados (format-date.pipe.ts)
│   │   │
│   │   ├── features/                   # Módulos de domínio de negócio (Feature-First)
│   │   │   ├── auth/                   # Autenticação
│   │   │   │   ├── auth-form.utils.ts  # Validações compartilhadas pelos formulários
│   │   │   │   ├── login/              # Tela de Login (Standalone Component)
│   │   │   │   └── register/           # Tela de Cadastro
│   │   │   │
│   │   │   ├── dashboard/              # Painel com indicadores das tarefas
│   │   │   │   └── dashboard-page.component.ts
│   │   │   │
│   │   │   ├── tasks/                  # Gestão de Tarefas
│   │   │   │   ├── components/         # Dumb components (task-card, status-badge)
│   │   │   │   ├── pages/              # Smart components (task-board-page, task-list-page, task-detail-page, task-form-page)
│   │   │   │   ├── services/           # TaskDataService (comunicação REST com service-task)
│   │   │   │   ├── models/             # Task, TaskStatus, TaskPayload
│   │   │   │   └── tasks.routes.ts     # Tabela de rotas lazy-loaded da feature
│   │   │   │
│   │   │   └── notifications/          # Tela de Atividades (eventos e saúde do consumidor)
│   │   │       └── notifications.component.ts
│   │   │
│   │   ├── app.config.ts               # Provedores globais, rotas e interceptores da aplicação
│   │   ├── app.routes.ts               # Tabela principal de rotas (lazy-loaded features)
│   │   └── app.component.ts            # Componente shell/raiz da aplicação
│   │
│   ├── assets/                         # Imagens, ícones estáticos e fontes
│   ├── environments/                   # URLs relativas das APIs
│   └── styles.scss                     # Estilos globais da aplicação
│
├── Dockerfile                          # Multi-Stage Build (Node.js -> NGINX)
├── angular.json
└── package.json
```

### Consequências

* Features isoladas por domínio, com `core/` e `shared/` livres de regra de negócio.
* `shared/` não importa tipos ou serviços de `features/`; componentes específicos de tarefas, como `StatusBadgeComponent`,
  permanecem em `features/tasks/components/`.
* Standalone Components e Signals dispensam `NgModules`, reduzindo boilerplate.
* Lazy loading por feature limita o bundle inicial.
* Componentes dumb permanecem reutilizáveis; páginas smart concentram estado e chamadas HTTP.

### Confirmação

* `npm run build` passa com `strict` e `strictTemplates`.
* `grep -R NgModule src` não encontra módulos.
* A estrutura de pastas e os interceptores conferem com a árvore acima.
* Rotas lazy-loaded: `/login`, `/register`, `/dashboard`, `/tasks`, `/tasks/board`, `/tasks/new`, `/tasks/:id`,
  `/tasks/:id/edit` e `/activities` (alias `/notifications`).
