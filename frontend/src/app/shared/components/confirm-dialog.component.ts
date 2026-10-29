import { DOCUMENT } from '@angular/common';
import { AfterViewInit, Component, ElementRef, HostListener, inject, input, OnDestroy, output, viewChild } from '@angular/core';

@Component({
  selector: 'app-confirm-dialog',
  template: `
    <div class="overlay" animate.enter="overlay-enter" animate.leave="overlay-leave" (click)="onOverlay($event)">
      <div class="dialog" role="dialog" aria-modal="true" [attr.aria-labelledby]="titleId()" [attr.aria-describedby]="descriptionId()">
        <span class="dialog-icon" aria-hidden="true">!</span>
        <h2 [id]="titleId()">{{ title() }}</h2>
        <p [id]="descriptionId()" class="muted">{{ description() }}</p>
        <hr class="rule" />
        <div class="dialog-actions">
          <button #cancelButton class="btn btn-outline" type="button" (click)="cancel.emit()" [disabled]="pending()">
            Cancelar <span class="arrow" aria-hidden="true">↗</span>
          </button>
          <button #confirmButton class="btn btn-danger" type="button" (click)="confirm.emit()" [disabled]="pending()">
            {{ pending() ? 'Excluindo...' : confirmLabel() }} <span class="arrow" aria-hidden="true">↗</span>
          </button>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .overlay { position: fixed; inset: 0; z-index: 50; display: grid; place-items: center; padding: 22px 22px 22px 262px; background: rgba(24, 61, 53, 0.15); }
    .dialog { width: min(530px, 100%); display: grid; padding: 34px 29px 27px; background: var(--paper-raised); border-radius: 3px; }
    .dialog-icon { display: grid; place-items: center; width: 47px; height: 47px; border-radius: 3px; background: var(--danger-soft); color: var(--danger); font-size: 24px; }
    h2 { margin-top: 15px; font-size: 27px; }
    .muted { margin-top: 13px; max-width: 433px; font-size: 14px; line-height: 1.3; }
    .rule { margin-top: 45px; }
    .dialog-actions { display: grid; grid-template-columns: 208px 1fr; gap: 11px; margin-top: 22px; }
    .dialog-actions .btn { width: 100%; }
    .overlay-enter { animation: overlay-in 180ms ease-out; }
    .overlay-enter .dialog { animation: dialog-in 220ms ease-out; }
    .overlay-leave { animation: overlay-out 140ms ease-in; }
    @keyframes overlay-in { from { opacity: 0; } }
    @keyframes overlay-out { to { opacity: 0; } }
    @keyframes dialog-in { from { opacity: 0; transform: translateY(12px) scale(0.985); } }
    @media (max-width: 900px) {
      .overlay { padding: 22px; }
      .dialog-actions { grid-template-columns: 1fr 1fr; }
    }
    @media (prefers-reduced-motion: reduce) {
      .overlay-enter, .overlay-enter .dialog, .overlay-leave { animation: none; }
    }
  `],
})
export class ConfirmDialogComponent implements AfterViewInit, OnDestroy {
  private readonly document = inject(DOCUMENT);
  private readonly previousFocus = this.document.activeElement as HTMLElement | null;
  private readonly cancelButton = viewChild.required<ElementRef<HTMLButtonElement>>('cancelButton');
  private readonly confirmButton = viewChild.required<ElementRef<HTMLButtonElement>>('confirmButton');
  readonly title = input.required<string>();
  readonly description = input.required<string>();
  readonly pending = input(false);
  readonly confirmLabel = input('Confirmar');
  readonly titleId = input('confirm-title');
  readonly descriptionId = input('confirm-description');
  readonly cancel = output<void>();
  readonly confirm = output<void>();

  ngAfterViewInit(): void {
    this.cancelButton().nativeElement.focus();
  }

  ngOnDestroy(): void {
    this.previousFocus?.focus();
  }

  @HostListener('document:keydown', ['$event'])
  onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape' && !this.pending()) {
      this.cancel.emit();
      return;
    }
    if (event.key !== 'Tab') return;

    const first = this.cancelButton().nativeElement;
    const last = this.confirmButton().nativeElement;
    if (event.shiftKey && this.document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && this.document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  onOverlay(event: MouseEvent): void {
    if (event.target === event.currentTarget && !this.pending()) this.cancel.emit();
  }
}
