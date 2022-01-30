import ReactDOM from "react-dom";
import React, { useState, useEffect } from "react";
import { Flipper, Flipped } from "react-flip-toolkit";
// import shuffle from "lodash.shuffle";
import "./styles.css";

import Button from '@mui/material/Button';
import Grid from "@mui/material/Grid";
import ArrowForwardRoundedIcon from '@mui/icons-material/ArrowForwardRounded';

function compareDecimals(a, b) {
  if (a.count === b.count) return 0;

  return a.count < b.count ? -1 : 1;
}

const ListShuffler = () => {
  const [data, setData] = useState([])
  const [totalTweets, setTweets] = useState(0);
  const sortList = () => {
    let copy = data.slice();
    copy.sort(compareDecimals);
    setData(copy);
  };

  const lst2str = (data) => {
    let out = "";
    for (let i = 0; i < data.length; i++) {
      out += data[i].count.toString() + data[i].word.toString();
    }

    return out;
  }

  const updateData = () => {
    fetch('/api/words/top')
      .then(response => response.json())
      .then(data => {
        setTweets(data["total"])
        setData(data["words"])
      });
  }

  useEffect(() => {
    // TODO: Make this into a web socket
    setInterval(updateData, 1000);
  }, [])

  return (
    <div>
      <div className="table-wrapper">
        <Grid container justifyContent="flex-end" >
          <Grid item md={3}>
            <p
            style={{
              margin: "15px",
              textAlign: "left",
              fontFamily: "Consolas, monaco, monospace",
              fontSize: "14px",
              background: "white",
            }}
          >
            Total Tweets: {totalTweets.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
          </p>
          </Grid>
        </Grid>
      </div>
      <div id="shuffle">
        <Flipper flipKey={lst2str(data)}>
          <ul className="table-wrapper">
            {data.map(({wordScore, multiple, count, word}) => (
              <Flipped key={word} flipId={word}>
                <li className="list-item card">
                  <Grid
                    container
                    justifyContent="space-between"
                    alignItems="center"
                    spacing={2}
                  >
                    <Grid item md={6} xs={4} overflow="hidden">
                      <p
                        style={{
                          margin: 0,
                          textOverflow: "ellipsis",
                          overflow: "hidden",
                          width: "100%"
                        }}
                      >
                        {word}
                      </p>
                    </Grid>
                    <Grid item md={6} xs={8}>
                      <Grid container alignItems="center" justifyContent="flex-end" textAlign="right" spacing={2}>
                        <Grid item md={5}>
                            <p className="stats-p">{Math.round(wordScore * 1000) / 1000}</p>
                        </Grid>
                        <Grid item md={2}>
                          <p className="stats-p">{Math.round(multiple * 100) / 100}</p>
                        </Grid>
                        <Grid item md={2}>
                          <p className="stats-p">{count}</p>
                        </Grid>
                        <Grid item md={3}>
                          <Grid container justifyContent="center" alignItems="center">
                            <Button
                              variant="outlined"
                              size="large"
                              onClick={() => window.open('https://twitter.com/search?q=' + encodeURIComponent(word), '_')}
                            >
                              <ArrowForwardRoundedIcon />
                            </Button>
                          </Grid>
                        </Grid>
                      </Grid>
                    </Grid>
                  </Grid>
                </li>
              </Flipped>
            ))}
          </ul>
        </Flipper>
      </div>
    </div>
  );
};

ReactDOM.render(<ListShuffler />, document.querySelector("#root"));
