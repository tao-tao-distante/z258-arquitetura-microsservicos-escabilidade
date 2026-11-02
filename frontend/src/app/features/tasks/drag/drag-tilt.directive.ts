import { DOCUMENT } from '@angular/common';
import { Directive, ElementRef, inject, NgZone, OnDestroy, OnInit } from '@angular/core';
import { leverFor, RESTING_TILT, stepTilt, TiltState } from './tilt-spring';

@Directive({ selector: '[appDragTilt]' })
export class DragTiltDirective implements OnInit, OnDestroy {
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly zone = inject(NgZone);
  private readonly document = inject(DOCUMENT);

  private state: TiltState = RESTING_TILT;
  private lever = 1;
  private lifted = true;
  private running = false;
  private frame = 0;
  private pointerX: number | null = null;
  private lastX: number | null = null;
  private lastTime: number | null = null;

  private readonly onPointerMove = (event: MouseEvent): void => {
    if (this.pointerX === null) {
      this.pivot(event.clientX, event.clientY);
    }
    this.pointerX = event.clientX;
  };

  private readonly onPointerUp = (): void => {
    this.lifted = false;
  };

  private readonly tick = (time: number): void => {
    const view = this.document.defaultView;
    if (!this.running || !view) {
      return;
    }
    if (this.lastTime !== null) {
      const dt = (time - this.lastTime) / 1000;
      const pointerVelocityX =
        dt > 0 && this.pointerX !== null && this.lastX !== null ? (this.pointerX - this.lastX) / dt : 0;
      this.state = stepTilt(this.state, { pointerVelocityX, lever: this.lever, lifted: this.lifted }, dt);
      this.render();
    }
    this.lastTime = time;
    this.lastX = this.pointerX;
    this.frame = view.requestAnimationFrame(this.tick);
  };

  ngOnInit(): void {
    const view = this.document.defaultView;
    if (!view || view.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
      return;
    }
    this.zone.runOutsideAngular(() => {
      this.document.addEventListener('pointermove', this.onPointerMove, { passive: true });
      this.document.addEventListener('pointerup', this.onPointerUp);
      this.running = true;
      this.frame = view.requestAnimationFrame(this.tick);
    });
  }

  ngOnDestroy(): void {
    this.running = false;
    this.document.defaultView?.cancelAnimationFrame(this.frame);
    this.document.removeEventListener('pointermove', this.onPointerMove);
    this.document.removeEventListener('pointerup', this.onPointerUp);
  }

  private pivot(clientX: number, clientY: number): void {
    const rect = this.host.nativeElement.getBoundingClientRect();
    const originY = clientY - rect.top;
    this.host.nativeElement.style.transformOrigin = `${clientX - rect.left}px ${originY}px`;
    this.lever = leverFor(originY, rect.height);
  }

  private render(): void {
    const { angle, scale } = this.state;
    this.host.nativeElement.style.transform = `rotate(${angle.toFixed(3)}deg) scale(${scale.toFixed(4)})`;
  }
}
