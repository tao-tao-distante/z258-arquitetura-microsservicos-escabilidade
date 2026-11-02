import { TestBed } from '@angular/core/testing';
import { ConfirmDialogComponent } from './confirm-dialog.component';

describe('ConfirmDialogComponent', () => {
  it('closes with Escape when no operation is pending', () => {
    const fixture = TestBed.createComponent(ConfirmDialogComponent);
    fixture.componentRef.setInput('title', 'Excluir?');
    fixture.componentRef.setInput('description', 'Esta ação não pode ser desfeita.');
    const cancel = vi.fn();
    fixture.componentInstance.cancel.subscribe(cancel);
    fixture.detectChanges();

    fixture.componentInstance.onKeydown(new KeyboardEvent('keydown', { key: 'Escape' }));

    expect(cancel).toHaveBeenCalledOnce();
  });

  it('keeps the dialog open with Escape while pending', () => {
    const fixture = TestBed.createComponent(ConfirmDialogComponent);
    fixture.componentRef.setInput('title', 'Excluir?');
    fixture.componentRef.setInput('description', 'Esta ação não pode ser desfeita.');
    fixture.componentRef.setInput('pending', true);
    const cancel = vi.fn();
    fixture.componentInstance.cancel.subscribe(cancel);
    fixture.detectChanges();

    fixture.componentInstance.onKeydown(new KeyboardEvent('keydown', { key: 'Escape' }));

    expect(cancel).not.toHaveBeenCalled();
  });

  it('emits from buttons and overlay while trapping keyboard focus', () => {
    const fixture = TestBed.createComponent(ConfirmDialogComponent);
    fixture.componentRef.setInput('title', 'Excluir?');
    fixture.componentRef.setInput('description', 'Esta ação não pode ser desfeita.');
    const cancel = vi.fn();
    const confirm = vi.fn();
    fixture.componentInstance.cancel.subscribe(cancel);
    fixture.componentInstance.confirm.subscribe(confirm);
    fixture.detectChanges();
    const root = fixture.nativeElement as HTMLElement;
    const buttons = Array.from(root.querySelectorAll('button')) as HTMLButtonElement[];

    buttons[1].click();
    buttons[0].click();
    buttons[0].focus();
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true }));
    expect(document.activeElement).toBe(buttons[1]);
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab' }));
    expect(document.activeElement).toBe(buttons[0]);

    const overlay = root.querySelector('.overlay') as HTMLElement;
    overlay.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    (root.querySelector('.dialog') as HTMLElement).dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(confirm).toHaveBeenCalledOnce();
    expect(cancel).toHaveBeenCalledTimes(2);
  });
});
