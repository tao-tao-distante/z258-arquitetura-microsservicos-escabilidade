import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { AuthService } from '../services/auth.service';

export const jwtInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const token = auth.sessionToken();
  const protectedRequest = isApiRequest(req.url, environment.taskUrl) ||
    isApiRequest(req.url, `${environment.identityUrl}/me`);
  const request = token && protectedRequest
    ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })
    : req;

  return next(request).pipe(
    catchError((error: unknown) => {
      if (
        error instanceof HttpErrorResponse &&
        error.status === 401 &&
        !isAuthenticationRequest(req.url)
      ) {
        auth.clearSession();
        if (!router.url.startsWith('/login')) {
          const returnUrl = isInternalUrl(router.url) ? router.url : '/tasks';
          void router.navigate(['/login'], { queryParams: { returnUrl } });
        }
      }
      return throwError(() => error);
    }),
  );
};

function isApiRequest(url: string, api: string): boolean {
  const path = new URL(url, location.origin).pathname;
  return path === api || path.startsWith(`${api}/`);
}

function isAuthenticationRequest(url: string): boolean {
  return isApiRequest(url, `${environment.identityUrl}/login`) ||
    isApiRequest(url, `${environment.identityUrl}/register`);
}

function isInternalUrl(url: string): boolean {
  return url.startsWith('/') && !url.startsWith('//') && !url.includes('\\');
}
