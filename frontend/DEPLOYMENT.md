# Frontend Deployment Guide - Vercel

**Platform:** Vercel
**Cost:** FREE (100GB bandwidth)
**Deploy Time:** 30 seconds
**Difficulty:** Beginner-friendly

---

## 🚀 Quick Setup (5 minutes)

### Step 1: Connect to Vercel

1. Go to https://vercel.com
2. Click "Sign Up" and choose "Continue with GitHub"
3. Click "Import Project"
4. Select your `howlerops` repository
5. Click "Import"

### Step 2: Configure Project

Vercel should auto-detect Vite. Verify these settings:

```
Framework Preset: Vite
Root Directory: frontend
Build Command: npm run build
Output Directory: dist
Install Command: npm install
```

### Step 3: Add Environment Variable

**IMPORTANT:** Add your backend URL as an environment variable.

In Vercel dashboard:
1. Go to **Settings → Environment Variables**
2. Add variable:
   - **Name:** `VITE_API_URL`
   - **Value:** `https://your-backend-url.run.app` (from backend deployment)
   - **Environment:** Production, Preview, Development (select all)

### Step 4: Deploy

Click **Deploy**

That's it! Vercel will:
- Build your frontend
- Deploy to global CDN
- Provide you with a URL like `https://howlerops.vercel.app`

---

## 🔄 Automatic Deployments

Once connected, Vercel automatically deploys:

- **Production:** Push to `main` branch → `https://howlerops.vercel.app`
- **Preview:** Pull request → `https://howlerops-git-feature-xyz.vercel.app`
- **Development:** Every push → preview URL

---

## 📝 Frontend Configuration

Update your frontend to use the environment variable:

```typescript
// src/lib/config.ts or similar
export const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8500'
```

Then use it in your API calls:
```typescript
fetch(`${API_URL}/api/auth/login`, {
  method: 'POST',
  // ...
})
```

---

## 🌐 Custom Domain (Optional)

1. Go to **Settings → Domains**
2. Add your domain (e.g., `app.yourdomain.com`)
3. Add DNS records as instructed by Vercel
4. Vercel automatically provisions SSL certificate

---

## 📊 Monitoring

Vercel provides:
- **Analytics:** Free for hobby projects
- **Error tracking:** Built-in
- **Deployment logs:** Every build
- **Preview URLs:** Automatic per-branch

Access at: https://vercel.com/dashboard

---

## 🔧 Vercel CLI (Optional)

For local deployments:

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy from terminal
cd frontend
vercel

# Deploy to production
vercel --prod
```

---

## ✅ Deployment Checklist

After deploying:

- [ ] Frontend is accessible at Vercel URL
- [ ] Backend API calls work (check Network tab)
- [ ] Environment variable `VITE_API_URL` is set
- [ ] CORS is configured in backend for Vercel domain
- [ ] Custom domain configured (optional)
- [ ] Team has access to Vercel dashboard

---

## 🔒 CORS Configuration

Update your backend to allow requests from Vercel:

In `backend-go/.env.production`:
```bash
ALLOWED_ORIGINS=https://howlerops.vercel.app,https://howlerops-git-*.vercel.app
```

Or in Cloud Run deployment, update allowed origins to include your Vercel domain.

---

## 💡 Tips

- **Preview deployments:** Every PR gets a unique URL
- **Rollback:** Instant rollback from Vercel dashboard
- **Environment variables:** Can be different per environment
- **Build logs:** Available in dashboard for debugging
- **Analytics:** Free analytics included

---

## 🆘 Troubleshooting

### Build Fails

**Check:**
1. `npm run build` works locally
2. All dependencies in `package.json`
3. Build logs in Vercel dashboard

### API Calls Fail

**Check:**
1. `VITE_API_URL` is set correctly
2. Backend CORS allows Vercel domain
3. Backend is deployed and healthy
4. Network tab in browser DevTools

### Environment Variables Not Working

**Vercel requires `VITE_` prefix:**
- ✅ `VITE_API_URL`
- ❌ `API_URL`

Rebuild after adding environment variables.

---

## 📚 Resources

- [Vercel Documentation](https://vercel.com/docs)
- [Vite Deployment Guide](https://vitejs.dev/guide/static-deploy.html)
- [Vercel Environment Variables](https://vercel.com/docs/concepts/projects/environment-variables)

---

**Deployment Platform:** Vercel
**Backend:** Google Cloud Run (see `backend-go/DEPLOYMENT_GUIDE.md`)
**Total Cost:** $0-3/month (with free tiers)
