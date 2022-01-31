import React, { useState, useEffect } from "react";
import Button from '@mui/material/Button';
import Grid from "@mui/material/Grid";
import ArrowForwardRoundedIcon from '@mui/icons-material/ArrowForwardRounded';
import { Base64 } from 'js-base64';
import { Flipped } from "react-flip-toolkit";

export const WordRow = (props) => {
  const [showButton, setShowButton] = useState(true);
  const [translationText, setTranslation] = useState("");
  const translate = () => {
    // the Base64 library has an encodeURI but it removes the padding for you with no option to keep it,
    // so I snipped this part of the source code and pasted it here, while keeping the padding
    let urlSafe = Base64.encode(props.word).replace(/[+\/]/g, function (m0) { return m0 == '+' ? '-' : '_'; });
    fetch('/api/translate?word=' + urlSafe)
      .then(response => response.json())
      .then(data => {
        // note: This isn't super necessary because it will be set the next time we download /api/words/top
        setTranslation("(" + data["translation"] + ")");
        setShowButton(false);
      });
    setShowButton(false);
  }
  useEffect(() => {
    if (props.translation) {
        setShowButton(false);
        setTranslation("(" + props.translation + ")");
    }
  })
  return (
    <Flipped key={props.word} flipId={props.word}>
        <li className="list-item card">
            <Grid
                container
                justifyContent="space-between"
                alignItems="center"
                spacing={2}
            >
            <Grid item md={5} xs={4} overflow="hidden">
            <p
                style={{
                margin: 0,
                textOverflow: "ellipsis",
                overflow: "hidden",
                width: "100%",
                display: "inline",
                marginRight: "5px"
                }}
            >
                {props.word}
            </p>
            <p className="translation">
                {translationText}
            </p>
            </Grid>
            <Grid item md={7} xs={8}>
                <Grid container alignItems="center" justifyContent="flex-end" textAlign="right" spacing={2}>
                    <Grid item md={3}>
                        {showButton && <Button
                        variant="outlined"
                        size="small"
                        onClick={translate}
                        >
                        Translate
                        </Button>}
                    </Grid>
                    <Grid item md={2}>
                        <p className="stats-p">{Math.round(props.wordScore * 1000) / 1000}</p>
                    </Grid>
                    <Grid item md={2}>
                        <p className="stats-p">{Math.round(props.multiple * 100) / 100}</p>
                    </Grid>
                    <Grid item md={2}>
                        <p className="stats-p">{props.count}</p>
                    </Grid>
                    <Grid item md={3}>
                        <Grid container justifyContent="center" alignItems="center">
                        <Button
                        variant="contained"
                        size="large"
                        onClick={() => window.open('https://twitter.com/search?q=' + encodeURIComponent(props.word), '_')}
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
  )
}