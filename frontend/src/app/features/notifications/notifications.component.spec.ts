import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NotificationsComponent } from './notifications.component';

describe('NotificationsComponent', () => {
  let fixture: ComponentFixture<NotificationsComponent>;
  let requests: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NotificationsComponent],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    requests = TestBed.inject(HttpTestingController);
  });

  afterEach(() => requests.verify());

  it('shows checking state before health responds and online after a 200 response', () => {
    fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.healthLabel()).toBe('VERIFICANDO');
    expect(fixture.componentInstance.healthTitle()).toBe('Verificando disponibilidade…');
    expect(fixture.componentInstance.tone()).toBe('doing');

    requests.expectOne('/api/notifications/health').flush(null, { status: 200, statusText: 'OK' });
    fixture.detectChanges();

    expect(fixture.componentInstance.healthLabel()).toBe('ONLINE');
    expect(fixture.componentInstance.healthTitle()).toBe('Consumidor ativo e saudável.');
    expect(fixture.componentInstance.tone()).toBe('done');
  });

  it('shows offline state for a non-200 response', () => {
    fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    requests.expectOne('/api/notifications/health').flush(null, { status: 202, statusText: 'Accepted' });

    expect(fixture.componentInstance.health()).toBe('offline');
    expect(fixture.componentInstance.healthLabel()).toBe('OFFLINE');
    expect(fixture.componentInstance.healthTitle()).toBe('Consumidor indisponível.');
    expect(fixture.componentInstance.tone()).toBe('todo');
  });

  it('shows offline state when health fails', () => {
    fixture = TestBed.createComponent(NotificationsComponent);
    fixture.detectChanges();
    requests.expectOne('/api/notifications/health').flush('failed', { status: 503, statusText: 'Unavailable' });

    expect(fixture.componentInstance.health()).toBe('offline');
  });
});
