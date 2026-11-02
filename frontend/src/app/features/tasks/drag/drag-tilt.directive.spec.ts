import { Component, NgZone } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DragTiltDirective } from './drag-tilt.directive';

@Component({
  imports: [DragTiltDirective],
  template: `<div appDragTilt></div>`,
})
class HostComponent {}

type FrameCallback = (time: number) => void;

describe('DragTiltDirective', () => {
  const originals = new Map<string, PropertyDescriptor | undefined>();
  let frames: FrameCallback[];
  let cancelled: number[];
  let scheduledInZone: boolean[];
  let reducedMotion: boolean;

  function replace(name: string, value: unknown): void {
    originals.set(name, Object.getOwnPropertyDescriptor(window, name));
    Object.defineProperty(window, name, { configurable: true, writable: true, value });
  }

  beforeEach(() => {
    frames = [];
    cancelled = [];
    scheduledInZone = [];
    reducedMotion = false;
    replace('requestAnimationFrame', (callback: FrameCallback) => {
      scheduledInZone.push(NgZone.isInAngularZone());
      frames.push(callback);
      return frames.length;
    });
    replace('cancelAnimationFrame', (id: number) => cancelled.push(id));
    replace('matchMedia', (query: string) => ({ matches: reducedMotion && query.includes('reduce') }));
  });

  afterEach(() => {
    for (const [name, descriptor] of originals) {
      if (descriptor) {
        Object.defineProperty(window, name, descriptor);
      } else {
        delete (window as unknown as Record<string, unknown>)[name];
      }
    }
    originals.clear();
  });

  function create(): { fixture: ComponentFixture<HostComponent>; host: HTMLElement } {
    const fixture = TestBed.createComponent(HostComponent);
    const host = fixture.nativeElement.querySelector('[appDragTilt]') as HTMLElement;
    host.getBoundingClientRect = () => ({ left: 0, top: 0, width: 200, height: 100 }) as DOMRect;
    fixture.detectChanges();
    return { fixture, host };
  }

  function pointer(type: string, clientX: number, clientY = 10): void {
    document.dispatchEvent(new MouseEvent(type, { clientX, clientY }));
  }

  function flush(time: number): void {
    const callback = frames.at(-1);
    callback?.(time);
  }

  function readTransform(host: HTMLElement): { angle: number; scale: number } {
    const match = /rotate\((-?[\d.e-]+)deg\) scale\(([\d.]+)\)/.exec(host.style.transform);
    expect(match).not.toBeNull();
    return { angle: Number(match![1]), scale: Number(match![2]) };
  }

  it('pivots on the grab point and swings while the pointer drags right', () => {
    const { host } = create();

    pointer('pointermove', 100, 10);
    flush(0);
    for (let step = 1; step <= 10; step++) {
      pointer('pointermove', 100 + step * 8, 10);
      flush(step * 16);
    }

    expect(host.style.transformOrigin).toBe('100px 10px');
    expect(readTransform(host).angle).toBeGreaterThan(0);
    expect(readTransform(host).scale).toBeGreaterThan(1);
  });

  it('runs the animation loop outside the Angular zone', () => {
    create();

    flush(0);
    flush(16);

    expect(scheduledInZone.length).toBeGreaterThan(1);
    expect(scheduledInZone.every((inZone) => !inZone)).toBe(true);
  });

  it('settles and lands the card after the pointer is released', () => {
    const { host } = create();
    pointer('pointermove', 100);
    flush(0);
    pointer('pointermove', 160);
    flush(16);

    pointer('pointerup', 160);
    for (let step = 2; step <= 200; step++) {
      flush(step * 16);
    }

    expect(Math.abs(readTransform(host).angle)).toBeLessThan(0.05);
    expect(readTransform(host).scale).toBeCloseTo(1, 3);
  });

  it('stops the loop and listeners when destroyed', () => {
    const { fixture, host } = create();
    flush(0);
    const lastFrame = frames.length;

    fixture.destroy();
    host.style.transform = '';
    pointer('pointermove', 300);

    expect(cancelled).toContain(lastFrame);
    expect(host.style.transform).toBe('');
    expect(host.style.transformOrigin).toBe('');
  });

  it('keeps the card still for users who prefer reduced motion', () => {
    reducedMotion = true;
    const { host } = create();

    pointer('pointermove', 100);

    expect(frames).toHaveLength(0);
    expect(host.style.transform).toBe('');
  });
});
