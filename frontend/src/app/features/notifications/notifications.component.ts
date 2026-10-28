import { Component, inject, OnInit, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';

@Component({
  selector: 'app-notifications',
  template: `
    <section class="stack">
      <div>
        <h1 style="margin: 0;">Notificações</h1>
        <p class="muted" style="margin: 0.25rem 0 0;">
          Eventos assíncronos processados pelo service-notification via RabbitMQ.
        </p>
      </div>

      <div class="card row" style="justify-content: space-between;">
        <div>
          <strong>Consumidor AMQP</strong>
          <p class="muted" style="margin: 0.25rem 0 0;">
            Fila <code>task.events</code> · DLQ habilitada · correlação por
            <code>x-correlation-id</code>
          </p>
        </div>
        <span
          class="pill"
          [class.up]="health() === 'online'"
          [class.down]="health() === 'offline'"
          role="status"
          aria-live="polite"
        >
          {{ healthLabel() }}
        </span>
      </div>

      <div class="card stack">
        <h2 style="margin: 0; font-size: 1rem;">Disponibilidade do serviço</h2>
        <p class="muted" style="margin: 0;">
          Esta tela consulta apenas o endpoint de saúde. O backend atual não fornece histórico ou feed
          de notificações para exibição.
        </p>
      </div>
    </section>
  `,
  styles: [
    `
      .pill {
        border-radius: 999px;
        padding: 0.25rem 0.7rem;
        font-size: 0.75rem;
        font-weight: 700;
        border: 1px solid var(--border);
      }
      .pill.up { color: var(--ok); }
      .pill.down { color: var(--danger); }
      code { color: var(--accent); }
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
    return { checking: 'verificando', online: 'online', offline: 'offline' }[this.health()];
  }
}
