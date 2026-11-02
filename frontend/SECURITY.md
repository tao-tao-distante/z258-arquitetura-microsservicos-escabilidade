# Segurança do frontend

O backend autentica exclusivamente por JWT no header `Authorization`; por isso, o token permanece no
`localStorage`. Cookies `HttpOnly` não são adotados enquanto o backend não oferecer esse contrato.

Esse armazenamento mantém risco residual de roubo do token em caso de XSS. As mitigações atuais são:

- Content Security Policy restritiva no NGINX (`script-src 'self'`, sem scripts inline ou origens externas);
- templates Angular com escape padrão, sem renderização de HTML não confiável;
- JWT enviado somente para rotas sob `/api/tasks` e `/api/identity/me` (perfil e troca de senha);
- token malformado ou expirado removido ao iniciar a aplicação;
- sessão removida após resposta HTTP 401 de endpoint protegido;
- troca de senha exige a senha atual e tem rate limit de 10 req/min/IP; senha atual incorreta responde `403`, sem
  derrubar a sessão.

A troca de senha não revoga tokens já emitidos: por ser stateless (ADR-0007), um token roubado antes da troca segue
válido até expirar (TTL de 1 hora).

A migração futura para cookie `HttpOnly`, `Secure` e `SameSite` exige primeiro suporte explícito do backend,
incluindo proteção CSRF.
