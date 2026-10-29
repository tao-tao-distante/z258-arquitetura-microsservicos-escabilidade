import { Component, inject, OnInit, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';

@Component({
  selector: 'app-notifications',
  template: `
    <section class="page">
      <header>
        <h1 class="page-title">Atividades</h1>
        <p class="page-subtitle">O que o sistema processou em segundo plano.</p>
      </header>

      <div class="grid">
        <section class="card health">
          <div class="health-head">
            <span class="eyebrow">CONSUMIDOR AMQP</span>
            <span [class]="'status-band ' + tone()" role="status" aria-live="polite">
              {{ healthLabel() }}
            </span>
          </div>
          <h2>{{ healthTitle() }}</h2>
          <p class="muted">
            Fila <code>task.events</code> · DLQ <code>task.events.dlq</code> · correlação por
            <code>x-correlation-id</code>
          </p>
        </section>

        <section class="card pipeline">
          <span class="eyebrow">PIPELINE</span>
          <ol>
            <li><span>01</span> Tarefa criada no service-task</li>
            <li><span>02</span> Evento gravado na outbox</li>
            <li><span>03</span> Publicação no RabbitMQ</li>
            <li><span>04</span> Consumo e confirmação</li>
          </ol>
        </section>
      </div>

      <section class="card note">
        <span class="eyebrow">NOTA</span>
        <p class="muted">
          Esta tela consulta apenas o endpoint de saúde do service-notification. O backend atual não
          expõe histórico ou feed de eventos para exibição.
        </p>
      </section>
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 26px; }
      .grid { display: grid; grid-template-columns: 1.4fr 1fr; gap: 16px; align-items: start; }
      .health { display: grid; gap: 12px; padding: 26px; }
      .health-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
      .health h2 { font-size: 24px; }
      .health .muted { font-size: 12.5px; line-height: 1.7; }
      code {
        font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
        font-size: 0.92em;
        background: #eef0e8;
        padding: 1px 5px;
        border-radius: 2px;
      }
      .pipeline { display: grid; gap: 14px; padding: 26px; }
      .pipeline ol { list-style: none; margin: 0; padding: 0; display: grid; gap: 10px; }
      .pipeline li { display: flex; gap: 12px; font-size: 12.5px; color: var(--muted); }
      .pipeline li span {
        font-size: 10px;
        letter-spacing: 0.06em;
        color: var(--pine);
        min-width: 18px;
      }
      .note { display: grid; gap: 8px; padding: 22px 26px; }
      .note .muted { font-size: 12.5px; }
      @media (max-width: 900px) {
        .grid { grid-template-columns: 1fr; }
      }
    `,
  ],
})
export class NotificationsComponent implements OnInit {
  private readonly http = inject(HttpClient);

  readonly health = signal<'checking' | 'online' | 'offline'>('checking');

  ngOnInit(): void {
    this.http.get(`${environment.notificationUrl}/health`, { observe: 'response' }).subscribe({
      next: (res) => this.health.set(res.status === 200 ? 'online' : 'offline'),
      error: () => this.health.set('offline'),
    });
  }

  healthLabel(): string {
    return { checking: 'VERIFICANDO', online: 'ONLINE', offline: 'OFFLINE' }[this.health()];
  }

  healthTitle(): string {
    return {
      checking: 'Verificando disponibilidade…',
      online: 'Consumidor ativo e saudável.',
      offline: 'Consumidor indisponível.',
    }[this.health()];
  }

  tone(): string {
    return this.health() === 'online' ? 'done' : this.health() === 'offline' ? 'todo' : 'doing';
  }
}
