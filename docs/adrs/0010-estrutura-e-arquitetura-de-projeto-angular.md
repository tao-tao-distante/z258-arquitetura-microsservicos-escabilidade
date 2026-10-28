# 0010 - Padronização de Arquitetura de Pastas e Layout do Projeto Angular Frontend

## Status

Aceito.

## Data

2026-10-28

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
│   │   │   ├── interceptors/          # Interceptores HTTP (correlation-id.interceptor.ts, jwt.interceptor.ts)
│   │   │   ├── services/               # Serviços globais (auth.service.ts)
│   │   │   └── models/                 # Interfaces/Types do ecossistema global
│   │   │
│   │   ├── shared/                     # Componentes visuais utilitários e reutilizáveis
│   │   │   ├── components/             # UI genérica (button, modal, navbar, loading-spinner)
│   │   │   ├── directives/             # Diretivas genéricas de DOM
│   │   │   └── pipes/                  # Pipes customizados (format-date.pipe.ts)
│   │   │
│   │   ├── features/                   # Módulos de domínio de negócio (Feature-First)
│   │   │   ├── auth/                   # Módulo de Autenticação
│   │   │   │   ├── login/              # Tela de Login (Standalone Component)
│   │   │   │   └── register/           # Tela de Cadastro
│   │   │   │
│   │   │   ├── tasks/                  # Módulo de Gestão de Tarefas
│   │   │   │   ├── components/         # Dumb components (task-card, task-filter)
│   │   │   │   ├── pages/              # Smart components (task-board-page, task-list-page)
│   │   │   │   ├── services/           # TaskDataService (comunicação REST com service-task)
│   │   │   │   ├── models/             # Task, Project, Status interfaces
│   │   │   │   └── tasks.routes.ts     # Tabela de rotas com Lazy Loading
│   │   │   │
│   │   │   └── notifications/          # Módulo de Notificações/Central de Avisos
│   │   │
│   │   ├── app.config.ts               # Provedores globais, rotas e interceptores da aplicação
│   │   ├── app.routes.ts               # Tabela principal de rotas (Lazy-loaded features)
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
* Standalone Components e Signals dispensam `NgModules`, reduzindo boilerplate.
* Lazy loading por feature limita o bundle inicial.
* Componentes dumb permanecem reutilizáveis; páginas smart concentram estado e chamadas HTTP.

### Confirmação

* `npm run build` passa com `strict` e `strictTemplates`.
* `grep -R NgModule src` não encontra módulos.
* A estrutura de pastas e os interceptores conferem com a árvore acima.
