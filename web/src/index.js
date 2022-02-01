import ReactDOM from "react-dom";
import React, { useState, useEffect } from "react";
import { Flipper } from "react-flip-toolkit";
// import shuffle from "lodash.shuffle";

import "./styles.css";
import { WordRow } from "./WordRow";
import Button from '@mui/material/Button';
import Grid from "@mui/material/Grid";
import ArrowForwardRoundedIcon from '@mui/icons-material/ArrowForwardRounded';
import Box from '@mui/material/Box';


const App = () => {
  const [data, setData] = useState([])
  const [totalTweets, setTweets] = useState(0);

  const lst2str = (data) => {
    let out = "";
    for (let i = 0; i < data.length; i++) {
      out += data[i].count.toString() + data[i].word.toString();
    }

    return out;
  }

  const updateData = () => {
    try {
      fetch('/api/words/top')
      .then(response => response.json())
      .then(data => {
        setTweets(data["total"])
        setData(data["words"])
      });
    } catch(err) {
      console.log(err)
    }
  }

  // this is a terrible way of updating translations and is 100% coupled with WordRow.
  // However, react-flip-toolkit has forced me to do it this way.
  const updateTranslation = (word, translation) => {
    let data2 = data.slice()
    for (let i = 0; i < data2.length; i++) {
      if (data2[i].word == word) {
        data2[i].translation = translation;
      }
    }

    setData(data2);
  }

  useEffect(() => {
    // TODO: Make this into a web socket
    updateData();
    setInterval(updateData, 1000);
  }, [])

  return (
    <div className="table-wrapper">
      <div>
        <Grid container justifyContent="flex-end" >
          <Grid item md={9}>
            <img src="/favicon.ico" style={{height: "64px", width: "64px", float: "left", marginTop: "5px"}} />
            <h2 className="big">Top Tweets</h2>
            <p className="sub-big">An Unbiased World View</p>
          </Grid>
          <Grid item md={3}>
            <p className="total-tweets">
            Total Tweets: {totalTweets.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
          </p>
          </Grid>
        </Grid>
        <Grid container className="explanation">
          <p>
            Top Tweets cuts through the noise of modern social media using algorithms and raw data. No political influences, no watch time incentives and no products to sell.
            Just an empircal source of truth for the world's events, in real time.
          </p>
        </Grid>
        <Grid container className="gradient">
          <Grid item md={12}>
            <h5 style={{marginTop: "1rem", marginBottom: 0}}>Word Score</h5>
            <span className="explanation">
              <p>
                Each word is given a score based on the Top Tweets algorithm that shows its level of interest and relevance to world events.
                The better the score, the more people care about a particular word.
              </p>
            </span>
          </Grid>
          <Grid item md={12}>
            <Box sx={{width: "100%", height: "1rem", background: 'linear-gradient(to right, #704141, #ff0000)'}}></Box>
          </Grid>
          <Grid item md={12}>
            <Grid container className="gradient-grades">
              <Grid item md={2} textAlign="left">
                <p>D</p>
              </Grid>
              <Grid item md={2} textAlign="center">
                <p>C</p>
              </Grid>
              <Grid item md={4} textAlign="center">
                <p>B</p>
              </Grid>
              <Grid item md={2} textAlign="center">
                <p>A</p>
              </Grid>
              <Grid item md={2} textAlign="right">
                <p>A+</p>
              </Grid>
            </Grid>
          </Grid>

        </Grid>
      </div>
      <div id="shuffle">
        <Flipper flipKey={lst2str(data)}>
          <ul className="shuffle-list">
            {data.map(({wordScore, multiple, count, word, translation}) => (
              <WordRow
                word={word}
                translation={translation}
                count={count}
                multiple={multiple}
                wordScore={wordScore}
                updateTranslation={updateTranslation}
                />
            ))}
          </ul>
        </Flipper>
      </div>
    </div>
  );
};

ReactDOM.render(<App />, document.querySelector("#root"));
