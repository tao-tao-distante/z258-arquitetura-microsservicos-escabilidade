# Segurança do frontend

O backend autentica exclusivamente por JWT no header `Authorization`; por isso, o token permanece no
`localStorage`. Cookies `HttpOnly` não são adotados enquanto o backend não oferecer esse contrato.

Esse armazenamento mantém risco residual de roubo do token em caso de XSS. As mitigações atuais são:

- Content Security Policy restritiva no NGINX (`script-src 'self'`, sem scripts inline ou origens externas);
- templates Angular com escape padrão, sem renderização de HTML não confiável;
- JWT enviado somente para rotas sob `/api/tasks`;
- token malformado ou expirado removido ao iniciar a aplicação;
- sessão removida após resposta HTTP 401 de endpoint protegido.

A migração futura para cookie `HttpOnly`, `Secure` e `SameSite` exige primeiro suporte explícito do backend,
incluindo proteção CSRF.
