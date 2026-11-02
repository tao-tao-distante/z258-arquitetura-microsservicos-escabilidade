import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AuthLayoutComponent } from './auth-layout.component';
import { FormFieldComponent } from './form-field.component';
import { PageHeaderComponent } from './page-header.component';

describe('shared presentation components', () => {
  it('renders both auth layout variants and optional footer', async () => {
    await TestBed.configureTestingModule({ imports: [AuthLayoutComponent] }).compileComponents();
    const fixture = TestBed.createComponent(AuthLayoutComponent);
    fixture.componentRef.setInput('panelLines', ['Organize', 'Conclua']);
    fixture.componentRef.setInput('panelDescription', 'Descrição');
    fixture.componentRef.setInput('meta', 'Meta');
    fixture.componentRef.setInput('variant', 'register');
    fixture.componentRef.setInput('showPanelFooter', false);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.panel.register')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.panel-foot')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Organize');

    fixture.componentRef.setInput('variant', 'login');
    fixture.componentRef.setInput('showPanelFooter', true);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.panel-foot')).not.toBeNull();
  });

  it('uses the desktop label as mobile fallback', async () => {
    await TestBed.configureTestingModule({ imports: [FormFieldComponent] }).compileComponents();
    const fixture: ComponentFixture<FormFieldComponent> = TestBed.createComponent(FormFieldComponent);
    fixture.componentRef.setInput('label', 'E-mail');
    fixture.componentRef.setInput('inputId', 'email');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('label').htmlFor).toBe('email');
    expect(fixture.nativeElement.querySelector('.mobile-field-label').textContent).toBe('E-mail');

    fixture.componentRef.setInput('mobileLabel', 'Seu e-mail');
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.mobile-field-label').textContent).toBe('Seu e-mail');
  });

  it('renders desktop and mobile page headings with action visibility', async () => {
    await TestBed.configureTestingModule({ imports: [PageHeaderComponent] }).compileComponents();
    const fixture = TestBed.createComponent(PageHeaderComponent);
    fixture.componentRef.setInput('title', 'Tarefas');
    fixture.componentRef.setInput('subtitle', 'Organize');
    fixture.componentRef.setInput('mobileEyebrow', 'WORKSPACE');
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.mobile-copy h1').textContent).toBe('Tarefas');
    expect(fixture.nativeElement.querySelector('.header-action').classList).toContain('mobile-hidden');

    fixture.componentRef.setInput('mobileTitle', 'Hoje');
    fixture.componentRef.setInput('showMobileAction', true);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.mobile-copy h1').textContent).toBe('Hoje');
    expect(fixture.nativeElement.querySelector('.header-action').classList).not.toContain('mobile-hidden');
  });
});
