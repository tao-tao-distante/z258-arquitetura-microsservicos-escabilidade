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
});
