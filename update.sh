cd web
npm run build
cd ..
docker compose build top_tweets
docker push calderwhite/top-tweets
