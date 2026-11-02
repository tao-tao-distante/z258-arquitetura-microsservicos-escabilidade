import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { AppComponent } from './app.component';
import { AuthUser } from './core/models/auth.model';
import { AuthService } from './core/services/auth.service';

describe('AppComponent', () => {
  const auth = { user: signal<AuthUser | null>(null), logout: vi.fn() };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideRouter([{ path: '**', children: [] }]),
        { provide: AuthService, useValue: auth },
      ],
    }).compileComponents();
  });

  it.each([
    [{ id: 1, email: 'user@example.com', name: 'Ana Souza' }, 'AS'],
    [{ id: 1, email: 'user@example.com', name: 'Bia' }, 'BI'],
    [{ id: 1, email: 'user@example.com', name: '' }, 'US'],
  ])('derives the account initials from %o', (user, initials) => {
    auth.user.set(user);
    const app = TestBed.createComponent(AppComponent).componentInstance;

    expect(app.initials()).toBe(initials);
  });

  it('links the account block and the mobile nav to the profile page', async () => {
    auth.user.set({ id: 1, email: 'user@example.com', name: 'Ana' });
    const fixture = TestBed.createComponent(AppComponent);
    await TestBed.inject(Router).navigateByUrl('/profile');
    fixture.detectChanges();
    const root: HTMLElement = fixture.nativeElement;

    expect(root.querySelector('a.account')?.getAttribute('href')).toBe('/profile');
    expect(root.querySelector('.mobile-nav a[href="/profile"]')?.textContent).toContain('PERFIL');
    expect(root.querySelector('.crumb')?.textContent).toBe('WORKSPACE / CONTA / PERFIL');
  });

  it('lists the profile as the first item of the rail and mobile menus', async () => {
    auth.user.set({ id: 1, email: 'user@example.com', name: 'Ana' });
    const fixture = TestBed.createComponent(AppComponent);
    await TestBed.inject(Router).navigateByUrl('/dashboard');
    fixture.detectChanges();
    const root: HTMLElement = fixture.nativeElement;
    const rail = Array.from(root.querySelectorAll('.rail-nav a'));
    const mobile = Array.from(root.querySelectorAll('.mobile-nav a'));

    expect(rail[0]?.getAttribute('href')).toBe('/profile');
    expect(rail[0]?.querySelector('span:not(.index)')?.textContent).toBe('Perfil');
    expect(rail.map((a) => a.querySelector('.index')?.textContent)).toEqual(['01', '02', '03', '04', '05']);
    expect(mobile[0]?.getAttribute('href')).toBe('/profile');
    expect(mobile[0]?.textContent?.trim()).toBe('PERFIL');
  });
});
